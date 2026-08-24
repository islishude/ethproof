package proof

import "testing"

func FuzzResolveStorageSlotsNeverPanics(f *testing.F) {
	f.Add([]byte(`{"storage":[{"label":"value","slot":"0","type":"uint"}],"types":{"uint":{"encoding":"inplace","label":"uint256","numberOfBytes":"32"}}}`), "value")
	f.Add([]byte(`{}`), "value[0]")
	f.Fuzz(func(t *testing.T, raw []byte, query string) {
		layout, err := ParseStorageLayoutJSON(raw, "", StorageLayoutFormatLayout)
		if err == nil {
			_, _ = ResolveStorageSlots(layout, query)
		}
	})
}
