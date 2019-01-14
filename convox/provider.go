package convox

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"

	"github.com/convox/rack/pkg/structs"
	"github.com/convox/rack/sdk"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	homedir "github.com/mitchellh/go-homedir"
)

const (
	// DefaultHost is the default value for which Convox rack host to connect to
	DefaultHost   = "console.convox.com"
	clientVersion = "20190103234204"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"host": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("CONVOX_HOST", ""),
				Description: "The convox host to use. If omitted 'host' will default to\n" +
					"the first nonempty value in: \n" +
					"[\"$CONVOX_RACK\" environment variable, \"config_path/host\" file,\"" + DefaultHost + "\"]",
			},
			"password": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("CONVOX_PASSWORD", ""),
			},
			"rack": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"config_path": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: configDefaultFunc(),
			},
		},
		DataSourcesMap: map[string]*schema.Resource{},
		ResourcesMap: map[string]*schema.Resource{
			"convox_app":           ResourceConvoxApp(UnpackRackClient),
			"convox_syslog":        ResourceConvoxSyslog(UnpackRackClient),
			"convox_resource_link": ResourceConvoxResourceLink(UnpackRackClient),
		},
		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	host := getHost(d)
	pass := getPassword(d, host)
	rack := d.Get("rack").(string)

	c, err := sdk.New(fmt.Sprintf("https://convox:%s@%s", url.QueryEscape(pass), host))
	if err != nil {
		return nil, err
	}
	c.Rack = rack

	return c, nil
}

func configDefaultFunc() schema.SchemaDefaultFunc {
	return func() (interface{}, error) {
		var path string
		var err error
		if path = os.Getenv("CONVOX_CONFIG"); path != "" {
			return path, nil
		}

		if path, err = homedir.Dir(); err != nil {
			log.Println("[WARNING] Failed to retrieve home directory:", err)
		}

		path = filepath.Join(path, ".convox")
		stat, err := os.Stat(path)
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}

		if stat != nil && !stat.IsDir() {
			return "", fmt.Errorf("Error expected config_path to be a directory")
		}
		return path, nil

	}
}

// UnpackRackClient casts the meta as a convox Client and specifies the rack from schema
func UnpackRackClient(d ValueGetter, meta interface{}) (structs.Provider, error) {
	if meta == nil {
		return nil, fmt.Errorf("meta is nil")
	}

	c := meta.(structs.Provider)
	if c == nil {
		return nil, fmt.Errorf("Could not convert meta to rack Client: %#v", meta)
	}

	return c, nil
}

func getHost(d *schema.ResourceData) string {
	if v := d.Get("host").(string); v != "" {
		return v
	}

	root := d.Get("config_path").(string)
	host, _ := ioutil.ReadFile(filepath.Join(root, "host"))
	if host != nil {
		return string(host)
	}
	return DefaultHost
}

func getPassword(d *schema.ResourceData, host string) string {
	if v := d.Get("password").(string); v != "" {
		return v
	}

	root := d.Get("config_path").(string)
	b, _ := ioutil.ReadFile(filepath.Join(root, "auth"))
	if b == nil {
		return ""
	}
	var auth map[string]string
	err := json.Unmarshal(b, &auth)
	if err != nil {
		return ""
	}
	return auth[host]
}
