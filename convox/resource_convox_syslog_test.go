package convox_test

import (
	"github.com/convox/rack/pkg/structs"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mattatcha/terraform-provider-convox/convox"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("ResourceConvoxSyslog", func() {
	convoxClient := &structs.MockProvider{}
	var unpacker convox.ClientUnpacker = func(valueGetter convox.ValueGetter, meta interface{}) (structs.Provider, error) {
		return convoxClient, nil
	}

	var resourceData *schema.ResourceData

	BeforeEach(func() {
		convoxClient = &structs.MockProvider{}

		resourceData = convox.ResourceConvoxSyslog(unpacker).Data(&terraform.InstanceState{
			Attributes: map[string]string{
				"name":     "test",
				"hostname": "logs.foo.com",
				"port":     "12345",
				"scheme":   "tcp+tls",
				"private":  "true",
			},
		})
	})

	Describe("CreateFunc", func() {
		var cut schema.CreateFunc
		BeforeEach(func() {
			cut = convox.ResourceConvoxSyslogCreateFactory(unpacker)
		})

		Describe("#ResourceConvoxSyslogCreateFactory", func() {
			It("should create the CreateFunc", func() {
				Expect(cut).ToNot(BeNil())
			})
		})

		Describe("creating the resource", func() {

			BeforeEach(func() {
				convoxClient.On("ResourceGet", mock.Anything).Return(&structs.Resource{
					Status: "running",
				}, nil)

				convoxClient.On("ResourceCreate", mock.Anything, mock.Anything).Return(&structs.Resource{
					Name:   "test",
					Status: "running",
					Url:    "tcp://1.1.1.1:1111",
				}, nil)

				Expect(cut(resourceData, resourceData)).To(BeNil())
			})

			It("should set the URL", func() {
				Expect(resourceData.Get("url")).To(Equal("tcp+tls://logs.foo.com:12345"))
			})

			It("should call the convox API with the right kind", func() {
				Expect(convoxClient.Calls[0].Arguments.String(0)).To(Equal("syslog"))
			})

			Describe("Options", func() {
				var opts structs.ResourceCreateOptions

				JustBeforeEach(func() {
					opts = convoxClient.Calls[0].Arguments[1].(structs.ResourceCreateOptions)
				})

				It("should call the convox API with the right name", func() {
					Expect(*opts.Name).To(Equal("test"))
				})

				It("should call the convox API with the right URL", func() {
					Expect(opts.Parameters["Url"]).To(Equal("tcp+tls://logs.foo.com:12345"))
				})

				It("should call the convox API with the right Private value", func() {
					Expect(opts.Parameters["Private"]).To(Equal("true"))
				})
			})
		})
	})

	Describe("ReadFunc", func() {
		var cut schema.ReadFunc
		BeforeEach(func() {
			cut = convox.ResourceConvoxSyslogReadFactory(unpacker)
		})

		Describe("#ResourceConvoxSyslogReadFactory", func() {
			It("should create the ReadFunc", func() {
				Expect(cut).ToNot(BeNil())
			})
		})

		Describe("Reading state", func() {
			BeforeEach(func() {
				convoxClient.On("ResourceGet", mock.Anything).Return(&structs.Resource{
					Name:   "test",
					Status: "running",
					Url:    "tcp://192.168.1.23:4567",
				}, nil)

				Expect(cut(resourceData, resourceData)).To(BeNil())
			})

			It("should ask for the resource by name", func() {
				Expect(convoxClient.Calls[0].Arguments.String(0)).To(Equal("test"))
			})

			It("should read the URL", func() {
				Expect(resourceData.Get("url")).To(Equal("tcp://192.168.1.23:4567"))
			})
		})
	})

	Describe("ResourceConvoxSyslogUpdateFactory", func() {
		var cut schema.UpdateFunc

		BeforeEach(func() {
			cut = convox.ResourceConvoxSyslogUpdateFactory(unpacker)
		})

		It("should make one", func() {
			Expect(cut).ToNot(BeNil())
		})

		Describe("Updating", func() {
			var requestedName string
			var requestedOptions structs.ResourceUpdateOptions

			BeforeEach(func() {
				requestedName = ""

				convoxClient.On("ResourceGet", mock.Anything).Return(&structs.Resource{
					Status: "running",
				}, nil)

				convoxClient.On("ResourceUpdate", mock.Anything, mock.Anything).Return(&structs.Resource{
					Name:   "test",
					Status: "running",
					Url:    "tcp://192.168.1.23:4567",
				}, nil).Run(func(args mock.Arguments) {
					requestedName = args[0].(string)
					requestedOptions = args[1].(structs.ResourceUpdateOptions)
				})

				Expect(cut(resourceData, resourceData)).To(BeNil())
			})

			It("should ask to update the right resource", func() {
				Expect(requestedName).To(Equal("test"))
			})

			It("should call the convox API with the right URL", func() {
				Expect(requestedOptions.Parameters["Url"]).To(Equal("tcp+tls://logs.foo.com:12345"))
			})

			It("should call the convox API with the right Private value", func() {
				Expect(requestedOptions.Parameters["Private"]).To(Equal("true"))
			})
		})
	})

	Describe("ResourceConvoxSyslogDeleteFactory", func() {
		var cut schema.DeleteFunc

		BeforeEach(func() {
			cut = convox.ResourceConvoxSyslogDeleteFactory(unpacker)
		})

		It("should make one", func() {
			Expect(cut).ToNot(BeNil())
		})

		Describe("Deleting", func() {

			BeforeEach(func() {

				convoxClient.On("ResourceDelete", mock.Anything).Return(nil)

				Expect(cut(resourceData, resourceData)).To(BeNil())
			})

			It("should delete the right resource", func() {
				Expect(convoxClient.Calls[0].Arguments.String(0)).To(Equal("test"))
			})
		})
	})
})
