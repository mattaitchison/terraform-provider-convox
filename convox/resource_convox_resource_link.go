package convox

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/convox/rack/pkg/structs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func ResourceConvoxResourceLink(clientUnpacker ClientUnpacker) *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"app_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"resource_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
		Read:   ResourceConvoxResourceLinkReadFactory(clientUnpacker),
		Create: ResourceConvoxResourceLinkCreateFactory(clientUnpacker),
		Delete: ResourceConvoxResourceLinkDeleteFactory(clientUnpacker),
	}
}

func ResourceConvoxResourceLinkCreateFactory(clientUnpacker ClientUnpacker) schema.CreateFunc {
	var createFunc schema.CreateFunc
	createFunc = func(d *schema.ResourceData, meta interface{}) error {
		if clientUnpacker == nil {
			return errors.New("clientUnpacker is required")
		}

		c, err := clientUnpacker(d, meta)
		if err != nil {
			return fmt.Errorf("Error unpacking client in CreateFunc: %s", err.Error())
		}

		resourceName := d.Get("resource_name").(string)
		app := d.Get("app_name").(string)

		_, err = c.ResourceLink(resourceName, app)
		if err != nil {
			if strings.Contains(err.Error(), "UPDATE_IN_PROGRESS") {
				if err := waitForRunning(c, resourceName); err != nil {
					return fmt.Errorf(
						"Error waiting for resource link API to become available: %s", err)
				}
				return createFunc(d, meta)
			}
			return fmt.Errorf("Error calling CreateLink(%s, %s): %s", app, resourceName, err.Error())
		}

		if err := waitForRunning(c, resourceName); err != nil {
			return fmt.Errorf(
				"Error waiting for resource link to be created: %s", err)
		}

		d.SetId(fmt.Sprintf("%s-%s", resourceName, app))

		return nil
	}

	return createFunc
}

func ResourceConvoxResourceLinkDeleteFactory(clientUnpacker ClientUnpacker) schema.DeleteFunc {
	return func(d *schema.ResourceData, meta interface{}) error {
		if clientUnpacker == nil {
			return errors.New("clientUnpacker is required")
		}

		c, err := clientUnpacker(d, meta)
		if err != nil {
			return fmt.Errorf("Error unpacking client in DeleteFunc: %s", err.Error())
		}

		resourceName := d.Get("resource_name").(string)
		app := d.Get("app_name").(string)

		_, err = c.ResourceUnlink(resourceName, app)
		if err != nil {
			return fmt.Errorf("Error calling DeleteLink(%s, %s): %s", app, resourceName, err.Error())
		}

		if err := waitForRunning(c, resourceName); err != nil {
			return fmt.Errorf(
				"Error waiting for resource link to be deleted: %s", err)
		}

		return nil
	}
}

func waitForRunning(c structs.Provider, resourceName string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{"updating"},
		Target:  []string{"running"},
		Refresh: readResourceStateFunc(c, resourceName),
		Timeout: 10 * time.Minute,
		Delay:   5 * time.Second,
	}

	if _, err := stateConf.WaitForState(); err != nil {
		return err
	}

	return nil
}

func ResourceConvoxResourceLinkReadFactory(clientUnpacker ClientUnpacker) schema.ReadFunc {
	return func(d *schema.ResourceData, meta interface{}) error {
		resourceName := d.Get("resource_name").(string)
		app := d.Get("app_name").(string)

		d.SetId(fmt.Sprintf("%s-%s", resourceName, app))

		return nil
	}
}
