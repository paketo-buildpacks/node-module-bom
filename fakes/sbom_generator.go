package fakes

import "sync"

type SBOMGenerator struct {
	GenerateCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			WorkingDir string
			LayersDir  string
			LayerName  string
		}
		Returns struct {
			Error error
		}
		Stub func(string, string, string) error
	}
}

func (f *SBOMGenerator) Generate(param1 string, param2 string, param3 string) error {
	f.GenerateCall.mutex.Lock()
	defer f.GenerateCall.mutex.Unlock()
	f.GenerateCall.CallCount++
	f.GenerateCall.Receives.WorkingDir = param1
	f.GenerateCall.Receives.LayersDir = param2
	f.GenerateCall.Receives.LayerName = param3
	if f.GenerateCall.Stub != nil {
		return f.GenerateCall.Stub(param1, param2, param3)
	}
	return f.GenerateCall.Returns.Error
}
