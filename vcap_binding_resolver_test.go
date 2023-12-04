package httpd_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/httpd"
	"github.com/sclevine/spec"
)

func testVcapBindingResolver(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		resolver                 *httpd.VcapBindingResolver
		bindingGuid              string
		platformDir              string
		expectedHtpasswdContents string
	)

	it.Before(func() {
		resolver = &httpd.VcapBindingResolver{}
		bindingGuid = "deadbeef"
		expectedHtpasswdContents = "user:$apr1$4A7us6ta$b.nzSqqXF2vkP8YbfOaH61"

		var err error
		platformDir, err = os.MkdirTemp("", "")
		Expect(err).NotTo(HaveOccurred())

		t.Setenv("VCAP_SERVICES",
			fmt.Sprintf(`
{
	"htpasswd": [
		{
			"name": "htpasswd",
			"label": "htpasswd",
			"binding_guid": "%s",
			"credentials": {
				".htpasswd": "%s"
			}
		}
	]
}
`,
				bindingGuid,
				expectedHtpasswdContents,
			),
		)
	})

	it("parses VCAP_SERVICES to populate the Path and Entries fields of the service bindings ", func() {
		bindings, err := resolver.Resolve("htpasswd", "", platformDir)
		Expect(err).NotTo(HaveOccurred())
		Expect(bindings).To(HaveLen(1))

		binding := bindings[0]
		Expect(binding.Path).To(Equal(filepath.Join(platformDir, bindingGuid)))
		Expect(binding.Entries).To(HaveLen(1))
		Expect(binding.Entries).To(HaveKey(".htpasswd"))

		entry := binding.Entries[".htpasswd"]
		htpasswdContents, err := entry.ReadString()
		Expect(err).NotTo(HaveOccurred())

		Expect(htpasswdContents).To(Equal(expectedHtpasswdContents))
	})

	context("when there are multiple bindings", func() {
		var (
			label1, label2                                         string
			provider1, provider2                                   string
			bindingGuid1, bindingGuid2, bindingGuid3, bindingGuid4 string
		)

		it.Before(func() {
			label1 = "htpasswd"
			label2 = "other-service"

			provider1 = "provider-1"
			provider2 = "provider-2"

			bindingGuid1 = "deadbeef-1"
			bindingGuid2 = "deadbeef-2"
			bindingGuid3 = "deadbeef-3"
			bindingGuid4 = "deadbeef-4"

			t.Setenv("VCAP_SERVICES",
				fmt.Sprintf(`
{
	"htpasswd": [
		{
			"name": "htpasswd-1",
			"label": "%s",
			"provider": "%s",
			"binding_guid": "%s",
			"credentials": {
				".htpasswd": "user:password"
			}
		},
		{
			"name": "htpasswd-2",
			"label": "%s",
			"provider": "%s",
			"binding_guid": "%s",
			"credentials": {
				".htpasswd": "user:password"
			}
		},
		{
			"name": "htpasswd-3",
			"label": "%s",
			"provider": "%s",
			"binding_guid": "%s",
			"credentials": {
				".htpasswd": "user:password"
			}
		}
	],
	"other-service": [
		{
			"name": "other-service",
			"label": "%s",
			"binding_guid": "%s",
			"credentials": {}
		}
	]
}
`,
					label1, provider1, bindingGuid1,
					label1, provider1, bindingGuid2,
					label1, provider2, bindingGuid3,
					label2, bindingGuid4,
				),
			)
		})

		it("returns all bindings such that the label matches the type", func() {
			bindings, err := resolver.Resolve("htpasswd", "", platformDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(bindings).To(HaveLen(3))

			Expect(bindings[0].Path).To(Equal(filepath.Join(platformDir, bindingGuid1)))
			Expect(bindings[0].Entries).To(HaveLen(1))

			Expect(bindings[1].Path).To(Equal(filepath.Join(platformDir, bindingGuid2)))
			Expect(bindings[1].Entries).To(HaveLen(1))

			Expect(bindings[2].Path).To(Equal(filepath.Join(platformDir, bindingGuid3)))
			Expect(bindings[2].Entries).To(HaveLen(1))
		})

		context("when the provider is given", func() {
			it("also filters on the provider", func() {
				bindings, err := resolver.Resolve("htpasswd", "provider-1", platformDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(bindings).To(HaveLen(2))

				Expect(bindings[0].Path).To(Equal(filepath.Join(platformDir, bindingGuid1)))
				Expect(bindings[0].Entries).To(HaveLen(1))

				Expect(bindings[1].Path).To(Equal(filepath.Join(platformDir, bindingGuid2)))
				Expect(bindings[1].Entries).To(HaveLen(1))
			})
		})
	})

	context("error cases", func() {
		context("when VCAP_SERVICES is not set", func() {
			it.Before(func() {
				os.Unsetenv("VCAP_SERVICES")
			})

			it("returns an empty list of bindings", func() {
				bindings, err := resolver.Resolve("htpasswd", "", platformDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(bindings).To(BeEmpty())
			})
		})

		context("when VCAP_SERVICES is not valid json", func() {
			it.Before(func() {
				t.Setenv("VCAP_SERVICES", "{123")
			})

			it("returns an error", func() {
				_, err := resolver.Resolve("htpasswd", "", platformDir)
				Expect(err).To(HaveOccurred())
			})
		})

		context("when writing the credentials to disk fails", func() {
			it("returns an error", func() {
				platformDir = "/proc"
				_, err := resolver.Resolve("htpasswd", "", platformDir)
				Expect(err).To(HaveOccurred())
			})
		})
	})
}
