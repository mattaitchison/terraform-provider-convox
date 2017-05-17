package convox

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/convox/rack/client"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

// ResourceConvoxApp returns the Resource schema describing a Convox App
func ResourceConvoxApp(clientUnpacker ClientUnpacker) *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"rack": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"status": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"environment": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
			},
			"params": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				Computed: true,
			},
			"balancers": &schema.Schema{
				Type:     schema.TypeMap,
				Computed: true,
			},
		},
		Create: resourceConvoxAppCreateFactory(clientUnpacker),
		Read:   resourceConvoxAppReadFactory(clientUnpacker),
		Update: resourceConvoxAppUpdateFactory(clientUnpacker),
		Delete: resourceConvoxAppDeleteFactory(clientUnpacker),
	}
}

func resourceConvoxAppCreateFactory(clientUnpacker ClientUnpacker) schema.CreateFunc {
	return func(d *schema.ResourceData, meta interface{}) error {
		c, err := clientUnpacker(d, meta)
		if err != nil {
			return err
		}

		name := d.Get("name").(string)
		app, err := c.CreateApp(name)
		if err != nil {
			return fmt.Errorf(
				"Error creating app (%s): %s", name, err)
		}

		d.SetId(app.Name)
		stateConf := &resource.StateChangeConf{
			Pending: []string{"creating"},
			Target:  []string{"running"},
			Refresh: appRefreshFunc(c, app.Name),
			Timeout: 10 * time.Minute,
			Delay:   25 * time.Second,
		}

		if _, err = stateConf.WaitForState(); err != nil {
			return fmt.Errorf(
				"Error waiting for app (%s) to be created: %s", app.Name, err)
		}

		// and then run update to set it up
		return resourceConvoxAppUpdateFactory(clientUnpacker)(d, meta)
	}
}

func resourceConvoxAppReadFactory(clientUnpacker ClientUnpacker) schema.ReadFunc {
	return func(d *schema.ResourceData, meta interface{}) error {
		c, err := clientUnpacker(d, meta)
		if err != nil {
			return err
		}

		app, err := c.GetApp(d.Get("name").(string))
		if err != nil {
			return err
		}
		d.SetId(app.Name)
		d.Set("release", app.Release)
		d.Set("status", app.Status)

		params, err := c.ListParameters(app.Name)
		if err != nil {
			return err
		}
		d.Set("params", params)

		env, err := c.GetEnvironment(app.Name)
		if err != nil {
			return err
		}
		d.Set("environment", env)

		formation, err := c.ListFormation(app.Name)
		if err != nil {
			return errwrap.Wrapf("Error while reading formation from Convox API: {{err}}", err)
		}
		return readFormation(d, formation)
	}
}

func resourceConvoxAppUpdateFactory(clientUnpacker ClientUnpacker) schema.UpdateFunc {
	return func(d *schema.ResourceData, meta interface{}) error {
		d.Partial(true)

		c, err := clientUnpacker(d, meta)
		if err != nil {
			return err
		}

		if err := setParams(c, d); err != nil {
			return err
		}
		d.SetPartial("params")

		if err := setEnv(c, d); err != nil {
			return err
		}
		d.SetPartial("environment")

		d.Partial(false)

		return resourceConvoxAppReadFactory(clientUnpacker)(d, meta)
	}
}

func resourceConvoxAppDeleteFactory(clientUnpacker ClientUnpacker) schema.DeleteFunc {
	return func(d *schema.ResourceData, meta interface{}) error {
		c, err := clientUnpacker(d, meta)
		if err != nil {
			return err
		}
		_, err = c.DeleteApp(d.Id())
		return err
	}
}

func readFormation(d *schema.ResourceData, v client.Formation) error {
	balancers := make(map[string]string, len(v))

	for _, f := range v {
		balancers[f.Name] = f.Balancer
	}

	if err := d.Set("balancers", balancers); err != nil {
		return errwrap.Wrapf("Unable to store balancers from formation: {{err}}", err)
	}

	return nil
}

func setParams(c Client, d *schema.ResourceData) error {
	if !d.HasChange("params") {
		return nil
	}

	raw := d.Get("params").(map[string]interface{})
	params := make(client.Parameters)
	for key := range raw {
		params[key] = raw[key].(string)
	}

	if err := c.SetParameters(d.Id(), params); err != nil {
		return fmt.Errorf("Error setting params (%#v) for %s: %s", params, d.Id(), err)
	}

	return nil
}

func setEnv(c Client, d *schema.ResourceData) error {
	if !d.HasChange("environment") {
		return nil
	}

	env := d.Get("environment").(map[string]interface{})
	data := ""
	for key, value := range env {
		data += fmt.Sprintf("%s=%s\n", key, value)
	}
	_, rel, err := c.SetEnvironment(d.Id(), strings.NewReader(data))
	if err != nil {
		return fmt.Errorf("Error setting vars (%#v) for %s: %s", env, d.Id(), err)
	}
	log.Printf("[INFO] Release (%s) created on: %s", rel, d.Id())

	return nil
}

func appRefreshFunc(client Client, name string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		app, err := client.GetApp(name)
		if err != nil {
			return app, "", err
		}
		return app, app.Status, err
	}
}
