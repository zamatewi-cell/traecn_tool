package device

import (
	"runtime"
	"strings"
	"testing"

	"github.com/zamatewi-cell/traecn_tool/internal/config"
)

func TestDeviceInfo_NewDeviceInfo(t *testing.T) {
	di := NewDeviceInfo()
	if di == nil {
		t.Fatal("NewDeviceInfo() returned nil")
	}
	if di.DeviceID == "" {
		t.Error("NewDeviceInfo() DeviceID is empty")
	}
	if di.MachineID == "" {
		t.Error("NewDeviceInfo() MachineID is empty")
	}
	if di.DeviceBrand == "" {
		t.Error("NewDeviceInfo() DeviceBrand is empty")
	}
	if di.DeviceCPU == "" {
		t.Error("NewDeviceInfo() DeviceCPU is empty")
	}
	if di.OSVersion == "" {
		t.Error("NewDeviceInfo() OSVersion is empty")
	}
	if di.DeviceType == "" {
		t.Error("NewDeviceInfo() DeviceType is empty")
	}
}

func TestDeviceInfo_Headers(t *testing.T) {
	di := NewDeviceInfo()
	headers := di.Headers()

	if headers == nil {
		t.Fatal("Headers() returned nil")
	}

	// Check required headers
	requiredHeaders := []string{
		"x-app-id",
		"x-app-version",
		"x-ide-version-code",
		"x-app-version-code",
		"x-device-brand",
		"x-device-cpu",
		"x-device-id",
		"x-machine-id",
		"x-os-version",
		"x-device-type",
		"x-ide-version",
		"x-ide-version-type",
		"request-traffic-type",
	}

	for _, header := range requiredHeaders {
		if _, exists := headers[header]; !exists {
			t.Errorf("Headers() missing required header: %s", header)
		}
	}

	// Verify header values
	if headers["x-app-id"] != config.AppID {
		t.Errorf("Headers() x-app-id = %v, want %v", headers["x-app-id"], config.AppID)
	}
	if headers["x-device-brand"] != di.DeviceBrand {
		t.Errorf("Headers() x-device-brand = %v, want %v", headers["x-device-brand"], di.DeviceBrand)
	}
	if headers["x-device-cpu"] != di.DeviceCPU {
		t.Errorf("Headers() x-device-cpu = %v, want %v", headers["x-device-cpu"], di.DeviceCPU)
	}
	if headers["x-device-id"] != di.DeviceID {
		t.Errorf("Headers() x-device-id = %v, want %v", headers["x-device-id"], di.DeviceID)
	}
	if headers["x-machine-id"] != di.MachineID {
		t.Errorf("Headers() x-machine-id = %v, want %v", headers["x-machine-id"], di.MachineID)
	}
	if headers["x-os-version"] != di.OSVersion {
		t.Errorf("Headers() x-os-version = %v, want %v", headers["x-os-version"], di.OSVersion)
	}
	if headers["x-device-type"] != di.DeviceType {
		t.Errorf("Headers() x-device-type = %v, want %v", headers["x-device-type"], di.DeviceType)
	}
}

func TestGenerateDeviceID(t *testing.T) {
	id1 := generateDeviceID()
	id2 := generateDeviceID()

	if id1 == "" {
		t.Error("generateDeviceID() returned empty string")
	}
	if id2 == "" {
		t.Error("generateDeviceID() returned empty string (second call)")
	}

	// Device ID should be 16 digits (between 1000000000000000 and 9999999999999999)
	if len(id1) != 16 {
		t.Errorf("generateDeviceID() length = %v, want 16", len(id1))
	}

	// Should be numeric
	for _, c := range id1 {
		if c < '0' || c > '9' {
			t.Errorf("generateDeviceID() contains non-numeric character: %c", c)
			break
		}
	}
}

func TestGenerateMachineID(t *testing.T) {
	mid1 := generateMachineID("test-hostname")
	mid2 := generateMachineID("test-hostname")
	mid3 := generateMachineID("different-hostname")

	if mid1 == "" {
		t.Error("generateMachineID() returned empty string")
	}

	// Same input should produce same output
	if mid1 != mid2 {
		t.Error("generateMachineID() not deterministic")
	}

	// Different input should produce different output
	if mid1 == mid3 {
		t.Error("generateMachineID() different inputs produced same output")
	}

	// Should be 64 characters (SHA256 hex)
	if len(mid1) != 64 {
		t.Errorf("generateMachineID() length = %v, want 64", len(mid1))
	}
}

func TestDetectCPU(t *testing.T) {
	cpu := detectCPU()
	if cpu == "" {
		t.Error("detectCPU() returned empty string")
	}

	// Should match architecture
	if runtime.GOARCH == "amd64" {
		if cpu != "Intel" {
			t.Errorf("detectCPU() = %v, want 'Intel' for amd64", cpu)
		}
	} else {
		expected := strings.ToUpper(runtime.GOARCH)
		if cpu != expected {
			t.Errorf("detectCPU() = %v, want %v", cpu, expected)
		}
	}
}

func TestDetectOS(t *testing.T) {
	os := detectOS()
	if os == "" {
		t.Error("detectOS() returned empty string")
	}

	// Should match runtime.GOOS
	switch runtime.GOOS {
	case "windows":
		if os != "Windows" {
			t.Errorf("detectOS() = %v, want 'Windows'", os)
		}
	case "darwin":
		if os != "macOS" {
			t.Errorf("detectOS() = %v, want 'macOS'", os)
		}
	case "linux":
		if os != "Linux" {
			t.Errorf("detectOS() = %v, want 'Linux'", os)
		}
	default:
		if os != runtime.GOOS {
			t.Errorf("detectOS() = %v, want %v", os, runtime.GOOS)
		}
	}
}

func TestDeviceInfo_Consistency(t *testing.T) {
	di1 := NewDeviceInfo()
	di2 := NewDeviceInfo()

	// DeviceID should be different for each instance
	if di1.DeviceID == di2.DeviceID {
		t.Error("NewDeviceInfo() generated same DeviceID for different instances")
	}

	// MachineID should be same (based on hostname)
	if di1.MachineID != di2.MachineID {
		t.Error("NewDeviceInfo() generated different MachineID for same hostname")
	}
}

func TestDeviceInfo_Headers_MultipleCalls(t *testing.T) {
	di := NewDeviceInfo()
	headers1 := di.Headers()
	headers2 := di.Headers()

	// Headers should be consistent
	for key, value1 := range headers1 {
		if value2, exists := headers2[key]; !exists {
			t.Errorf("Headers() second call missing key: %s", key)
		} else if value1 != value2 {
			t.Errorf("Headers() inconsistent value for %s: %v vs %v", key, value1, value2)
		}
	}
}
