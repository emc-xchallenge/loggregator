// This file was generated by counterfeiter
package fakeclient

import (
	"metron/clientpool"
	"sync"
)

type FakeClient struct {
	SchemeStub        func() string
	schemeMutex       sync.RWMutex
	schemeArgsForCall []struct{}
	schemeReturns     struct {
		result1 string
	}
	AddressStub        func() string
	addressMutex       sync.RWMutex
	addressArgsForCall []struct{}
	addressReturns     struct {
		result1 string
	}
	WriteStub        func([]byte) (int, error)
	writeMutex       sync.RWMutex
	writeArgsForCall []struct {
		arg1 []byte
	}
	writeReturns struct {
		result1 int
		result2 error
	}
	CloseStub        func() error
	closeMutex       sync.RWMutex
	closeArgsForCall []struct{}
	closeReturns     struct {
		result1 error
	}
}

func (fake *FakeClient) Scheme() string {
	fake.schemeMutex.Lock()
	fake.schemeArgsForCall = append(fake.schemeArgsForCall, struct{}{})
	fake.schemeMutex.Unlock()
	if fake.SchemeStub != nil {
		return fake.SchemeStub()
	} else {
		return fake.schemeReturns.result1
	}
}

func (fake *FakeClient) SchemeCallCount() int {
	fake.schemeMutex.RLock()
	defer fake.schemeMutex.RUnlock()
	return len(fake.schemeArgsForCall)
}

func (fake *FakeClient) SchemeReturns(result1 string) {
	fake.SchemeStub = nil
	fake.schemeReturns = struct {
		result1 string
	}{result1}
}

func (fake *FakeClient) Address() string {
	fake.addressMutex.Lock()
	fake.addressArgsForCall = append(fake.addressArgsForCall, struct{}{})
	fake.addressMutex.Unlock()
	if fake.AddressStub != nil {
		return fake.AddressStub()
	} else {
		return fake.addressReturns.result1
	}
}

func (fake *FakeClient) AddressCallCount() int {
	fake.addressMutex.RLock()
	defer fake.addressMutex.RUnlock()
	return len(fake.addressArgsForCall)
}

func (fake *FakeClient) AddressReturns(result1 string) {
	fake.AddressStub = nil
	fake.addressReturns = struct {
		result1 string
	}{result1}
}

func (fake *FakeClient) Write(arg1 []byte) (int, error) {
	fake.writeMutex.Lock()
	fake.writeArgsForCall = append(fake.writeArgsForCall, struct {
		arg1 []byte
	}{arg1})
	fake.writeMutex.Unlock()
	if fake.WriteStub != nil {
		return fake.WriteStub(arg1)
	} else {
		return fake.writeReturns.result1, fake.writeReturns.result2
	}
}

func (fake *FakeClient) WriteCallCount() int {
	fake.writeMutex.RLock()
	defer fake.writeMutex.RUnlock()
	return len(fake.writeArgsForCall)
}

func (fake *FakeClient) WriteArgsForCall(i int) []byte {
	fake.writeMutex.RLock()
	defer fake.writeMutex.RUnlock()
	return fake.writeArgsForCall[i].arg1
}

func (fake *FakeClient) WriteReturns(result1 int, result2 error) {
	fake.WriteStub = nil
	fake.writeReturns = struct {
		result1 int
		result2 error
	}{result1, result2}
}

func (fake *FakeClient) Close() error {
	fake.closeMutex.Lock()
	fake.closeArgsForCall = append(fake.closeArgsForCall, struct{}{})
	fake.closeMutex.Unlock()
	if fake.CloseStub != nil {
		return fake.CloseStub()
	} else {
		return fake.closeReturns.result1
	}
}

func (fake *FakeClient) CloseCallCount() int {
	fake.closeMutex.RLock()
	defer fake.closeMutex.RUnlock()
	return len(fake.closeArgsForCall)
}

func (fake *FakeClient) CloseReturns(result1 error) {
	fake.CloseStub = nil
	fake.closeReturns = struct {
		result1 error
	}{result1}
}

var _ clientpool.Client = new(FakeClient)
