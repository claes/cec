package cec

import(
	"log"
	"time"
	"strings"
)

type Device struct {
	logical_address int
	active_source bool
	power_status string
	vendor_id uint64 
	physical_address string
	osd_name string
}

var logicalNames = []string{"TV", "Recording", "Recording2", "Tuner", "Playback", "Audio", "Tuner2", "Tuner3", "Playback2", "Recording3", "Tuner4", "Playback3", "Reserved", "Reserved2", "Free", "Broadcast"}

func Open(name string, deviceName string) {
	var config CECConfiguration
	config.DeviceName = deviceName

	if er := cecInit(config); er != nil {
		log.Println(er)
		return	
	}

	adapter, er := getAdapter(name)
	if er != nil {
		log.Println(er)
		return
	}

	er = openAdapter(adapter)
	if er != nil {
		log.Println(er)
		return
	}
}

func Key(address int, key int) {
	er := KeyPress(address, key)
	if er != nil {
		log.Println(er)
		return
	}
	time.Sleep(10 * time.Millisecond)
	er = KeyRelease(address)
	if er != nil {
		log.Println(er)
		return
	}
}

func List() map[string]Device {
	devices := make(map[string]Device)

	active_devices := GetActiveDevices()

	for address, active := range active_devices {
		if (active) {
			var dev Device

			dev.logical_address = address
			dev.osd_name = GetDeviceOSDName(address)
			dev.power_status = GetDevicePowerStatus(address)
			dev.active_source = IsActiveSource(address)
			dev.physical_address = GetDevicePhysicalAddress(address)
			dev.vendor_id = GetDeviceVendorId(address)

			devices[logicalNames[address]] = dev
		}
	}

	return devices
}

func removeSeparators(in string) string {
        // remove separators (":", "-", " ", "_")
        out := strings.Map(func(r rune) rune {
                if strings.IndexRune(":-_ ", r) < 0 {
                        return r
                }
                return -1
        }, in)

	return(out)
}

func GetLogicalAddressByName(name string) int {
	name = removeSeparators(name)
	l := len(name)

	if name[l-1] == '1' {
		name = name[:l-1]
	}

	for i:=0; i<16; i++ {
		if logicalNames[i] == name {
			return i
		}
	}

	if name == "Unregistered" {
		return 15
	}

	return -1
}

func GetLogicalNameByAddress(addr int) string {
	return logicalNames[addr]
}
