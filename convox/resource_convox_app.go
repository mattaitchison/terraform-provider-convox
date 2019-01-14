package convox

import (
	"fmt"
	"strings"
	"time"

	"github.com/convox/rack/pkg/helpers"
	"github.com/convox/rack/pkg/structs"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

// ResourceConvoxApp returns the Resource schema describing a Convox App
func ResourceConvoxApp(clientUnpacker ClientUnpacker) *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"generation": &schema.Schema{
				Type:             schema.TypeString,
				Optional:         true,
				Default:          "1",
				ForceNew:         true,
				DiffSuppressFunc: generationDiffSuppress,
			},
			"status": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"environment": &schema.Schema{
				Type:             schema.TypeMap,
				Optional:         true,
				Sensitive:        true,
				DiffSuppressFunc: environmentDiffSuppress,
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
		Create: ResourceConvoxAppCreateFactory(clientUnpacker),
		Read:   ResourceConvoxAppReadFactory(clientUnpacker),
		Update: ResourceConvoxAppUpdateFactory(clientUnpacker),
		Delete: ResourceConvoxAppDeleteFactory(clientUnpacker),
	}
}

// ResourceConvoxAppCreateFactory builds the resource CreateFunc for a Convox App resource
func ResourceConvoxAppCreateFactory(clientUnpacker ClientUnpacker) schema.CreateFunc {
	if clientUnpacker == nil {
		panic("clientUnpacker is required")
	}

	return func(d *schema.ResourceData, meta interface{}) error {
		c, err := clientUnpacker(d, meta)
		if err != nil {
			return errwrap.Wrapf("Error unpacking Convox client in App CreateFunc: {{err}}", err)
		}

		name := d.Get("name").(string)
		generation := d.Get("generation").(string)
		if generation == "" {
			generation = "1"
		}

		options := structs.AppCreateOptions{
			Generation: &generation,
		}

		app, err := c.AppCreate(name, options)
		if err != nil {
			return errwrap.Wrapf(fmt.Sprintf(
				"Error creating app (%s): {{err}}", name), err)
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
			return errwrap.Wrapf(fmt.Sprintf("Error waiting for app (%s) to be created: {{err}}", app.Name), err)
		}

		// and then run update to set it up
		return ResourceConvoxAppUpdateFactory(clientUnpacker)(d, meta)
	}
}

// ResourceConvoxAppReadFactory create the ReadFunc for a Convox App resource
func ResourceConvoxAppReadFactory(clientUnpacker ClientUnpacker) schema.ReadFunc {
	if clientUnpacker == nil {
		panic("clientUnpacker is required")
	}

	return func(d *schema.ResourceData, meta interface{}) error {
		c, err := clientUnpacker(d, meta)
		if err != nil {
			return errwrap.Wrapf("Error unpacking Convox client in App ReadFunc: {{err}}", err)
		}

		app, err := c.AppGet(d.Get("name").(string))
		if err != nil {
			return err
		}
		d.SetId(app.Name)

		err = d.Set("status", app.Status)
		if err != nil {
			return fmt.Errorf("Error setting the status key: %s", err.Error())
		}

		params := app.Parameters
		err = d.Set("params", params)
		if err != nil {
			return errwrap.Wrapf("Error while setting params: {{err}}", err)
		}

		env, err := helpers.AppEnvironment(c, app.Name)
		if err != nil {
			return fmt.Errorf("Error calling GetEnvironment(%s): %s", app.Name, err.Error())
		}

		d.Set("environment", env)

		processes, err := c.ServiceList(app.Name)
		if err != nil {
			return errwrap.Wrapf("Error while reading formation from Convox API: {{err}}", err)
		}
		return readFormation(d, processes)
	}
}

// ResourceConvoxAppUpdateFactory builds the UpdateFunc for a Convox App resource
func ResourceConvoxAppUpdateFactory(clientUnpacker ClientUnpacker) schema.UpdateFunc {
	if clientUnpacker == nil {
		panic("clientUnpacker is required")
	}

	return func(d *schema.ResourceData, meta interface{}) error {
		d.Partial(true)
		name := d.Get("name").(string)

		c, err := clientUnpacker(d, meta)
		if err != nil {
			return errwrap.Wrapf("Error unpacking Convox client in App UpdateFunc: {{err}}", err)
		}

		if err := setParams(c, name, d); err != nil {
			return err
		}
		d.SetPartial("params")

		if err := setEnv(c, name, d); err != nil {
			return err
		}
		d.SetPartial("environment")

		d.Partial(false)

		return ResourceConvoxAppReadFactory(clientUnpacker)(d, meta)
	}
}

// ResourceConvoxAppDeleteFactory builds the DeleteFunc for a Convox App resource
func ResourceConvoxAppDeleteFactory(clientUnpacker ClientUnpacker) schema.DeleteFunc {
	if clientUnpacker == nil {
		panic("clientUnpacker is required")
	}

	return func(d *schema.ResourceData, meta interface{}) error {
		c, err := clientUnpacker(d, meta)
		if err != nil {
			return errwrap.Wrapf("Error unpacking Convox client in App DeleteFunc: {{err}}", err)
		}
		err = c.AppDelete(d.Id())
		return err
	}
}

func readFormation(d *schema.ResourceData, v structs.Services) error {
	balancers := make(map[string]string, len(v))

	for _, f := range v {
		balancers[f.Name] = f.Domain
	}

	if err := d.Set("balancers", balancers); err != nil {
		return errwrap.Wrapf("Unable to store balancers from formation: {{err}}", err)
	}

	return nil
}

func setParams(c structs.Provider, appName string, d *schema.ResourceData) error {
	if !d.HasChange("params") {
		return nil
	}

	raw := d.Get("params").(map[string]interface{})
	params := make(map[string]string)

	opts := structs.AppUpdateOptions{
		Parameters: map[string]string{},
	}
	for key := range raw {
		opts.Parameters[key] = raw[key].(string)
	}

	if err := c.AppUpdate(appName, opts); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Error setting params (%#v) for %s: {{err}}", params, d.Id()), err)
	}

	return nil
}

func environmentDiffSuppress(k, old, new string, d *schema.ResourceData) bool {
	// Return true if the diff should be suppressed, false to retain it.

	if d.IsNewResource() {
		return false
	}

	if old == new {
		return true
	}

	return strings.TrimSpace(old) == strings.TrimSpace(new)
}

func generationDiffSuppress(k, old, new string, d *schema.ResourceData) bool {
	// Return true if the diff should be suppressed, false to retain it.
	if old == "" && new == "1" {
		return true
	}

	return old == new
}

func setEnv(c structs.Provider, appName string, d *schema.ResourceData) error {
	if !d.HasChange("environment") {
		return nil
	}

	env := d.Get("environment").(map[string]interface{})
	data := ""
	for key, value := range env {
		data += fmt.Sprintf("%s=%s\n", key, value)
	}

	_, err := c.ReleaseCreate(appName, structs.ReleaseCreateOptions{Env: &data})
	if err != nil {
		return fmt.Errorf("Error setting vars (%#v) for %s: %s", env, d.Id(), err)
	}

	return nil
}

func appRefreshFunc(client structs.Provider, name string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		app, err := client.AppGet(name)
		if err != nil {
			return app, "", err
		}
		return app, app.Status, err
	}
}
