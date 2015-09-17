package udev

/*
#cgo pkg-config: gudev-1.0

#include <gudev/gudev.h>
*/
import "C"

import (
	"runtime"
	"unsafe"
)

type Client struct {
	p *C.struct__GUdevClient
}

type Device struct {
	p *C.struct__GUdevDevice
}

func New(subsystems []string) *Client {
	// convert go to char **
	cs := make([]*C.gchar, len(subsystems)+1)
	for i := range subsystems {
		cs[i] = (*C.gchar)(C.CString(subsystems[i]))
	}

	p := C.g_udev_client_new((**C.gchar)(unsafe.Pointer(&cs[0])))
	client := &Client{
		p: p,
	}
	runtime.SetFinalizer(client, func(p *Client) {
		C.g_object_unref((C.gpointer)(client.p))
	})
	return client
}

func (c *Client) QueryBySubsystem(subsystem string) []Device {
	l := C.g_udev_client_query_by_subsystem(c.p, (*C.gchar)(C.CString(subsystem)))
	result := make([]Device, C.g_list_length(l))
	for i := range result {
		p := (*C.struct__GUdevDevice)(l.data)
		device := Device{
			p: p,
		}
		runtime.SetFinalizer(&device, func(device *Device) {
			C.g_object_unref((C.gpointer)(device.p))
		})
		result[i] = device
		l = l.next
	}
	C.g_list_free(l)

	return result
}

func (d *Device) GetSysfsAttr(name string) string {
	res := C.g_udev_device_get_sysfs_attr(d.p, (*C.gchar)(C.CString(name)))
	return C.GoString((*C.char)(res))
}

func (d *Device) GetProperty(name string) string {
	res := C.g_udev_device_get_property(d.p, (*C.gchar)(C.CString(name)))
	return C.GoString((*C.char)(res))
}

func (d *Device) GetName() string {
	res := C.g_udev_device_get_name(d.p)
	return C.GoString((*C.char)(res))
}

func (d *Device) GetDeviceFile() string {
	res := C.g_udev_device_get_device_file(d.p)
	return C.GoString((*C.char)(res))
}

func (d *Device) GetParent() *Device {
	res := C.g_udev_device_get_parent(d.p)
	if res == nil {
		return nil
	}

	return &Device{p: res}
}
