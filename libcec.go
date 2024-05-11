package cec

/*
#cgo pkg-config: libcec
//#cgo CFLAGS: -Iinclude
//#cgo LDFLAGS: -lcec
#include <stdio.h>
#include <stdlib.h>
#include <libcec/cecc.h>
#include <stdint.h>

ICECCallbacks g_callbacks;
// callbacks.go exports
void logMessageCallback(void *, const cec_log_message *);
void commandReceived(void *, const cec_command *);
void keyPressed(void *, const cec_keypress *);
void alertReceived(void *, const libcec_alert, const libcec_parameter);

libcec_configuration * allocConfiguration()  {
	libcec_configuration * ret = (libcec_configuration*)malloc(sizeof(libcec_configuration));
	memset(ret, 0, sizeof(libcec_configuration));
	return ret;
}

void freeConfiguration(libcec_configuration * conf) {
	free(conf);
}

void setupCallbacks(libcec_configuration *conf)
{
	g_callbacks.logMessage = &logMessageCallback;
	g_callbacks.keyPress = &keyPressed;
	g_callbacks.commandReceived = &commandReceived;
	g_callbacks.configurationChanged = NULL;
	g_callbacks.alert = &alertReceived;
	g_callbacks.menuStateChanged = NULL;
	g_callbacks.sourceActivated = NULL;
	(*conf).callbacks = &g_callbacks;
}

void setName(libcec_configuration *conf, char *name)
{
	snprintf((*conf).strDeviceName, 13, "%s", name);
}

*/
import "C"

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"unsafe"
)

// Connection class
type Connection struct {
	connection C.libcec_connection_t
	Commands   chan *Command
	KeyPresses chan int
	Messages   chan string
}

type cecAdapter struct {
	Path string
	Comm string
}

func cecInit(c *Connection, deviceName string) (C.libcec_connection_t, error) {
	var connection C.libcec_connection_t
	var conf *C.libcec_configuration = C.allocConfiguration()
	defer C.freeConfiguration(conf)

	conf.clientVersion = C.uint32_t(C.LIBCEC_VERSION_CURRENT)
	conf.deviceTypes.types[0] = C.CEC_DEVICE_TYPE_RECORDING_DEVICE
	conf.callbackParam = unsafe.Pointer(c)
	conf.bActivateSource = 0
	conf.baseDevice = 0      // CECDEVICE_TV               = 0,
	conf.tvVendor = 0x0000F0 // Samsung
	//conf.iHDMIPort = 4
	//conf.iButtonReleaseDelayMs = 0
	//conf.iPhysicalAddress = 0
	C.setName(conf, C.CString(deviceName))
	C.setupCallbacks(conf)

	connection = C.libcec_initialise(conf)
	if connection == C.libcec_connection_t(nil) {
		return connection, errors.New("Failed to init CEC")
	}
	return connection, nil
}

func getAdapter(connection C.libcec_connection_t, name string) (cecAdapter, error) {
	var adapter cecAdapter

	var deviceList [10]C.cec_adapter
	devicesFound := int(C.libcec_find_adapters(connection, &deviceList[0], 10, nil))

	log.Printf("Found %d adapters: \n", devicesFound)
	for i := 0; i < devicesFound; i++ {
		log.Printf("%d: %v %v\n", i, deviceList[i].comm[0], deviceList[i].path[0])
	}

	for i := 0; i < devicesFound; i++ {
		device := deviceList[i]
		adapter.Path = C.GoStringN(&device.path[0], 1024)
		adapter.Comm = C.GoStringN(&device.comm[0], 1024)

		if strings.Contains(adapter.Path, name) || strings.Contains(adapter.Comm, name) {
			return adapter, nil
		}
	}

	return adapter, errors.New("No Device Found")
}

func openAdapter(connection C.libcec_connection_t, adapter cecAdapter) error {
	log.Println("libcec_init_video_standalone")
	C.libcec_init_video_standalone(connection)

	log.Println("libcec_open")
	result := C.libcec_open(connection, C.CString(adapter.Comm), C.CEC_DEFAULT_CONNECT_TIMEOUT)
	if result < 1 {
		return errors.New("Failed to open adapter")
	}

	return nil
}

// Transmit CEC command - command is encoded as a hex string with
// colons (e.g. "40:04")
func (c *Connection) Transmit(command string) {
	var cecCommand C.cec_command

	cmd, err := hex.DecodeString(removeSeparators(command))
	if err != nil {
		log.Fatal(err)
	}
	cmdLen := len(cmd)

	if cmdLen > 0 {
		cecCommand.initiator = C.cec_logical_address((cmd[0] >> 4) & 0xF)
		cecCommand.destination = C.cec_logical_address(cmd[0] & 0xF)
		if cmdLen > 1 {
			cecCommand.opcode_set = 1
			cecCommand.opcode = C.cec_opcode(cmd[1])
		} else {
			cecCommand.opcode_set = 0
		}
		if cmdLen > 2 {
			cecCommand.parameters.size = C.uint8_t(cmdLen - 2)
			for i := 0; i < cmdLen-2; i++ {
				cecCommand.parameters.data[i] = C.uint8_t(cmd[i+2])
			}
		} else {
			cecCommand.parameters.size = 0
		}
	}

	C.libcec_transmit(c.connection, (*C.cec_command)(&cecCommand))
}

// Destroy - destroy the cec connection
func (c *Connection) Destroy() {
	C.libcec_destroy(c.connection)
}

// PowerOn - power on the device with the given logical address
func (c *Connection) PowerOn(address int) error {
	if C.libcec_power_on_devices(c.connection, C.cec_logical_address(address)) != 0 {
		return errors.New("Error in cec_power_on_devices")
	}
	return nil
}

// Standby - put the device with the given address in standby mode
func (c *Connection) Standby(address int) error {
	if C.libcec_standby_devices(c.connection, C.cec_logical_address(address)) != 0 {
		return errors.New("Error in cec_standby_devices")
	}
	return nil
}

// VolumeUp - send a volume up command to the amp if present
func (c *Connection) VolumeUp() error {
	if C.libcec_volume_up(c.connection, 1) != 0 {
		return errors.New("Error in cec_volume_up")
	}
	return nil
}

// VolumeDown - send a volume down command to the amp if present
func (c *Connection) VolumeDown() error {
	if C.libcec_volume_down(c.connection, 1) != 0 {
		return errors.New("Error in cec_volume_down")
	}
	return nil
}

// Mute - send a mute/unmute command to the amp if present
func (c *Connection) Mute() error {
	if C.libcec_mute_audio(c.connection, 1) != 0 {
		return errors.New("Error in cec_mute_audio")
	}
	return nil
}

// KeyPress - send a key press (down) command code to the given address
func (c *Connection) KeyPress(address int, key int) error {
	if C.libcec_send_keypress(c.connection, C.cec_logical_address(address), C.cec_user_control_code(key), 1) != 1 {
		return errors.New("Error in cec_send_keypress")
	}
	return nil
}

// KeyRelease - send a key releas command to the given address
func (c *Connection) KeyRelease(address int) error {
	if C.libcec_send_key_release(c.connection, C.cec_logical_address(address), 1) != 1 {
		return errors.New("Error in cec_send_key_release")
	}
	return nil
}

// GetActiveDevices - returns an array of active devices
func (c *Connection) GetActiveDevices() [16]bool {
	var devices [16]bool
	result := C.libcec_get_active_devices(c.connection)

	for i := 0; i < 16; i++ {
		if int(result.addresses[i]) > 0 {
			devices[i] = true
		}
	}

	return devices
}

// GetDeviceOSDName - get the OSD name of the specified device
func (c *Connection) GetDeviceOSDName(address int) string {
	name := make([]byte, 14)
	C.libcec_get_device_osd_name(c.connection, C.cec_logical_address(address), (*C.char)(unsafe.Pointer(&name[0])))

	return string(name)
}

// IsActiveSource - check if the device at the given address is the active source
func (c *Connection) IsActiveSource(address int) bool {
	result := C.libcec_is_active_source(c.connection, C.cec_logical_address(address))

	if int(result) != 0 {
		return true
	}

	return false
}

// SetActiveSource
func (c *Connection) SetActiveSource(device_type int) bool {
	result := C.libcec_set_active_source(c.connection, C.cec_device_type(device_type))

	if int(result) != 0 {
		return true
	}

	return false
}

// RescanDevices
func (c *Connection) RescanDevices() {
	C.libcec_rescan_devices(c.connection)
}

// GetDeviceVendorID - Get the Vendor-ID of the device at the given address
func (c *Connection) GetDeviceVendorID(address int) uint64 {
	result := C.libcec_get_device_vendor_id(c.connection, C.cec_logical_address(address))

	return uint64(result)
}

// GetDevicePhysicalAddress - Get the physical address of the device at
// the given logical address
func (c *Connection) GetDevicePhysicalAddress(address int) string {
	result := C.libcec_get_device_physical_address(c.connection, C.cec_logical_address(address))

	return fmt.Sprintf("%x.%x.%x.%x", (uint(result)>>12)&0xf, (uint(result)>>8)&0xf, (uint(result)>>4)&0xf, uint(result)&0xf)
}

// Poll device - poll the device at
// the given logical address
func (c *Connection) PollDevice(address int) string {
	result := C.libcec_poll_device(c.connection, C.cec_logical_address(address))

	return fmt.Sprintf("%T: %+v", result, result)
}

//extern DECLSPEC int libcec_set_osd_string(libcec_connection_t connection, cec_namespace cec_logical_address ilogicaladdress, cec_namespace cec_display_control duration, const char* strmessage);
func (c *Connection) SetOSDString(address int, str string) error {
	msg := []byte(str)
	if C.libcec_set_osd_string(c.connection, C.cec_logical_address(address), C.cec_display_control(1), (*C.char)(unsafe.Pointer(&msg[0]))) != 0 {
		return errors.New("Error in cec_set_osd_string")
	}
	return nil
}

// GetDevicePowerStatus - Get the power status of the device at the
// given address
func (c *Connection) GetDevicePowerStatus(address int) string {
	result := C.libcec_get_device_power_status(c.connection, C.cec_logical_address(address))

	// C.CEC_POWER_STATUS_UNKNOWN == error

	if int(result) == C.CEC_POWER_STATUS_ON {
		return "on"
	} else if int(result) == C.CEC_POWER_STATUS_STANDBY {
		return "standby"
	} else if int(result) == C.CEC_POWER_STATUS_IN_TRANSITION_STANDBY_TO_ON {
		return "starting"
	} else if int(result) == C.CEC_POWER_STATUS_IN_TRANSITION_ON_TO_STANDBY {
		return "shutting down"
	} else {
		return ""
	}
}
