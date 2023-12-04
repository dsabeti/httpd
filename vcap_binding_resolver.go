package httpd

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit/v2/servicebindings"
)

type VcapBindingResolver struct{}

func NewVcapBindingResolver() *VcapBindingResolver {
	return &VcapBindingResolver{}
}

type VcapServices map[string][]VcapBinding

type VcapBinding struct {
	Name        string            `json:"name"`
	Label       string            `json:"label"`
	Provider    string            `json:"provider"`
	BindingGuid string            `json:"binding_guid"`
	Credentials map[string]string `json:"credentials"`
}

func (v *VcapBindingResolver) Resolve(typ, provider, workingDir string) ([]servicebindings.Binding, error) {
	envVar, ok := os.LookupEnv("VCAP_SERVICES")
	if !ok {
		return []servicebindings.Binding{}, nil
	}

	vcapServices, err := ParseVcapServices([]byte(envVar))
	if err != nil {
		return nil, err
	}

	serviceBindings, err := vcapServices.
		FilterOnTypeAndProvider(typ, provider).
		ToServiceBindings(workingDir)
	if err != nil {
		return nil, err
	}

	return serviceBindings, nil
}

func ParseVcapServices(contents []byte) (*VcapServices, error) {
	var vcapServices VcapServices
	err := json.Unmarshal(contents, &vcapServices)
	if err != nil {
		return nil, err
	}

	return &vcapServices, nil
}

func (s *VcapServices) FilterOnTypeAndProvider(typ, provider string) *VcapServices {
	filteredVcapServices := VcapServices{}

	for key, service := range *s {
		for _, binding := range service {
			if binding.Label == typ &&
				(provider == "" || provider == binding.Provider) {
				if bindingsList, ok := filteredVcapServices[key]; ok {
					filteredVcapServices[key] = append(bindingsList, binding)
				} else {
					filteredVcapServices[key] = []VcapBinding{binding}
				}
			}
		}
	}

	return &filteredVcapServices
}

func (s *VcapServices) ToServiceBindings(workingDir string) ([]servicebindings.Binding, error) {
	var svcBindings []servicebindings.Binding

	for _, vcapBindings := range *s {
		for _, vcapBinding := range vcapBindings {
			binding, err := vcapBinding.ToServiceBinding(workingDir)
			if err != nil {
				return nil, err
			}
			svcBindings = append(svcBindings, binding)
		}
	}

	return svcBindings, nil
}

func (b *VcapBinding) ToServiceBinding(workingDir string) (servicebindings.Binding, error) {
	bindingDirectory := filepath.Join(workingDir, b.BindingGuid)

	err := os.MkdirAll(bindingDirectory, 0755)
	if err != nil {
		return servicebindings.Binding{}, err
	}

	entries, err := b.credentialsToEntries(b.Credentials, bindingDirectory)
	if err != nil {
		return servicebindings.Binding{}, err
	}

	serviceBinding := servicebindings.Binding{
		Path:    bindingDirectory,
		Entries: entries,
	}

	return serviceBinding, nil
}

func (b *VcapBinding) credentialsToEntries(credentials map[string]string, bindingDirectory string) (map[string]*servicebindings.Entry, error) {
	var err error
	entries := make(map[string]*servicebindings.Entry)

	for key, credential := range credentials {
		filename := filepath.Join(bindingDirectory, key)

		err = os.WriteFile(filename, []byte(credential), 0777)
		if err != nil {
			return nil, err
		}

		entry := servicebindings.NewEntry(filename)
		entries[key] = entry
	}
	return entries, nil
}
