package convox

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/convox/rack/pkg/structs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

// ResourceConvoxSyslog describes the schema for a Convox Syslog Resource resource
func ResourceConvoxSyslog(clientUnpacker ClientUnpacker) *schema.Resource {
	log.Printf("ResourceConvoxSyslog - schema.Resource")
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"hostname": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"port": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"scheme": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateScheme,
			},
			"private": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"url": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
		Create: ResourceConvoxSyslogCreateFactory(clientUnpacker),
		Read:   ResourceConvoxSyslogReadFactory(clientUnpacker),
		Update: ResourceConvoxSyslogUpdateFactory(clientUnpacker),
		Delete: ResourceConvoxSyslogDeleteFactory(clientUnpacker),
	}
}

func validateScheme(value interface{}, key string) (warnings []string, errors []error) {
	v := value.(string)
	if v != "tcp" && v != "tcp+tls" {
		errors = []error{fmt.Errorf("`scheme` must be either 'tcp' or 'tcp+tls'")}
	}

	return
}

func formURLString(d *schema.ResourceData) string {
	return fmt.Sprintf("%s://%s:%d", d.Get("scheme"), d.Get("hostname"), d.Get("port"))
}

func readResourceStateFunc(c structs.Provider, resourceName string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resource, err := c.ResourceGet(resourceName)
		if err != nil {
			return resource, "", err
		}
		return resource, resource.Status, err
	}
}

// ResourceConvoxSyslogCreateFactory builds the Convox Syslog CreateFunc
func ResourceConvoxSyslogCreateFactory(clientUnpacker ClientUnpacker) schema.CreateFunc {
	log.Printf("ResourceConvoxSyslogCreateFactory")

	return func(d *schema.ResourceData, meta interface{}) error {
		log.Printf("CreateFunc")

		if clientUnpacker == nil {
			return errors.New("clientUnpacker is required")
		}

		c, err := clientUnpacker(d, meta)
		if err != nil {
			return fmt.Errorf("Error unpacking client in CreateFunc: %s", err.Error())
		}

		parameters := map[string]string{
			"Url": formURLString(d),
		}

		if v, ok := d.GetOk("private"); ok {
			parameters["Private"] = fmt.Sprintf("%v", v)
		}

		name := d.Get("name").(string)
		options := structs.ResourceCreateOptions{
			Name:       &name,
			Parameters: parameters,
		}
		log.Printf("[INFO] Calling Convox CreateResource...")
		_, err = c.ResourceCreate("syslog", options)
		if err != nil {
			return fmt.Errorf("Error calling CreateResource: %s -- %v", err.Error(), options)
		}

		stateConf := &resource.StateChangeConf{
			Pending: []string{"creating"},
			Target:  []string{"running"},
			Refresh: readResourceStateFunc(c, name),
			Timeout: 10 * time.Minute,
			Delay:   1 * time.Second,
		}

		if _, err = stateConf.WaitForState(); err != nil {
			return fmt.Errorf(
				"Error waiting for resource (%s) to be created: %s", name, err)
		}

		d.SetId(name)

		d.Set("url", parameters["Url"])

		return nil
	}
}

// ResourceConvoxSyslogReadFactory builds the ReadFunc for the Convox Syslog Resource
func ResourceConvoxSyslogReadFactory(clientUnpacker ClientUnpacker) schema.ReadFunc {
	return func(d *schema.ResourceData, meta interface{}) error {
		log.Printf("ReadFunc")

		if clientUnpacker == nil {
			return errors.New("clientUnpacker is required")
		}

		c, err := clientUnpacker(d, meta)
		if err != nil {
			return fmt.Errorf("Error getting client in ReadFunc: %s", err.Error())
		}

		name := d.Get("name").(string)
		log.Printf("[INFO] Calling Convox GetResource(%s)...", name)
		convoxResource, err := c.ResourceGet(name)
		if err != nil {
			return fmt.Errorf("Error calling GetResource: %s", err.Error())
		}

		d.SetId(name)
		d.Set("url", convoxResource.Url)

		return nil
	}
}

// ResourceConvoxSyslogUpdateFactory creates the UpdateFunc for the Convox Syslog Resource
func ResourceConvoxSyslogUpdateFactory(clientUnpacker ClientUnpacker) schema.UpdateFunc {
	return func(d *schema.ResourceData, meta interface{}) error {
		log.Printf("UpdateFunc")

		if clientUnpacker == nil {
			return errors.New("clientUnpacker is required")
		}

		c, err := clientUnpacker(d, meta)
		if err != nil {
			return fmt.Errorf("Error getting client in UpdateFunc: %s", err.Error())
		}

		parameters := map[string]string{
			"Url": formURLString(d),
		}

		if v, ok := d.GetOk("private"); ok {
			parameters["Private"] = fmt.Sprintf("%v", v)
		}

		options := structs.ResourceUpdateOptions{
			Parameters: parameters,
		}

		name := d.Get("name").(string)
		log.Printf("[INFO] Calling Convox UpdateResource(%s, <options>)...", name)
		_, err = c.ResourceUpdate(name, options)
		if err != nil {
			return fmt.Errorf("Error calling UpdateResource: %s -- %v", err.Error(), options)
		}

		stateConf := &resource.StateChangeConf{
			Pending: []string{"updating"},
			Target:  []string{"running"},
			Refresh: readResourceStateFunc(c, name),
			Timeout: 10 * time.Minute,
			Delay:   1 * time.Second,
		}

		if _, err = stateConf.WaitForState(); err != nil {
			return fmt.Errorf(
				"Error waiting for resource (%s) to be updated: %s", name, err)
		}

		d.Set("url", parameters["Url"])

		return nil
	}
}

// ResourceConvoxSyslogDeleteFactory builds the DeleteFunc for thw Convox Syslog Resource
func ResourceConvoxSyslogDeleteFactory(clientUnpacker ClientUnpacker) schema.DeleteFunc {
	return func(d *schema.ResourceData, meta interface{}) error {
		log.Printf("DeleteFunc")

		if clientUnpacker == nil {
			return errors.New("clientUnpacker is required")
		}

		c, err := clientUnpacker(d, meta)
		if err != nil {
			return fmt.Errorf("Error getting client in DeleteFunc: %s", err.Error())
		}

		name := d.Get("name").(string)
		log.Printf("[INFO] Calling Convox DeleteResource(%s)...", name)
		err = c.ResourceDelete(name)
		if err != nil {
			return fmt.Errorf("Error calling DeleteResource: %s", err.Error())
		}

		return nil
	}
}
