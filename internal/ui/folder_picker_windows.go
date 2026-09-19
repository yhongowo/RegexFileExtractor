//go:build windows && cgo

package ui

/*
#cgo LDFLAGS: -lole32 -lshell32 -luuid
#define COBJMACROS
#include <windows.h>
#include <shobjidl.h>
#include <shlobj.h>

// The common item dialog is the Explorer-style folder picker on Windows 10/11.
// The caller owns the returned path and must release it with CoTaskMemFree.
static HRESULT rfePickFolder(HWND owner, LPCWSTR initial, PWSTR *selected) {
	*selected = NULL;
	HRESULT hr = CoInitializeEx(NULL, COINIT_APARTMENTTHREADED | COINIT_DISABLE_OLE1DDE);
	if (FAILED(hr)) return hr;

	IFileOpenDialog *dialog = NULL;
	hr = CoCreateInstance(&CLSID_FileOpenDialog, NULL, CLSCTX_INPROC_SERVER,
		&IID_IFileOpenDialog, (void **)&dialog);
	if (SUCCEEDED(hr)) {
		FILEOPENDIALOGOPTIONS options;
		hr = IFileOpenDialog_GetOptions(dialog, &options);
		if (SUCCEEDED(hr)) {
			hr = IFileOpenDialog_SetOptions(dialog,
				options | FOS_PICKFOLDERS | FOS_FORCEFILESYSTEM);
		}
		if (SUCCEEDED(hr) && initial != NULL && initial[0] != L'\0') {
			IShellItem *folder = NULL;
			if (SUCCEEDED(SHCreateItemFromParsingName(initial, NULL,
				&IID_IShellItem, (void **)&folder))) {
				// Preserve Explorer's recent location after the first invocation.
				IFileOpenDialog_SetDefaultFolder(dialog, folder);
				IShellItem_Release(folder);
			}
		}
		if (SUCCEEDED(hr)) hr = IFileOpenDialog_Show(dialog, owner);
		if (SUCCEEDED(hr)) {
			IShellItem *result = NULL;
			hr = IFileOpenDialog_GetResult(dialog, &result);
			if (SUCCEEDED(hr)) {
				hr = IShellItem_GetDisplayName(result, SIGDN_FILESYSPATH, selected);
				IShellItem_Release(result);
			}
		}
		IFileOpenDialog_Release(dialog);
	}
	CoUninitialize();
	return hr;
}

static void rfeFreeFolderPath(PWSTR path) { CoTaskMemFree(path); }
*/
import "C"

import (
	"fmt"
	"os"
	"runtime"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"
	"golang.org/x/sys/windows"
)

func showNativeFolderPicker(window fyne.Window, initial string, done func(string, error)) bool {
	native, ok := window.(driver.NativeWindow)
	if !ok {
		return false
	}
	var hwnd uintptr
	native.RunNative(func(context any) {
		if win, ok := context.(driver.WindowsWindowContext); ok {
			hwnd = win.HWND
		}
	})
	if hwnd == 0 {
		return false
	}
	go func() {
		// COM apartment initialization and dialog lifetime must stay on one OS thread.
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		if info, err := os.Stat(initial); err != nil || !info.IsDir() {
			initial = ""
		}
		var start *uint16
		if initial != "" {
			start, _ = windows.UTF16PtrFromString(initial)
		}
		var selected C.PWSTR
		hr := C.rfePickFolder(C.HWND(unsafe.Pointer(hwnd)),
			(*C.WCHAR)(unsafe.Pointer(start)), &selected)
		var path string
		if selected != nil {
			path = windows.UTF16PtrToString((*uint16)(unsafe.Pointer(selected)))
			C.rfeFreeFolderPath(selected)
		}
		var err error
		if code := uint32(hr); int32(hr) < 0 && code != 0x800704c7 {
			err = fmt.Errorf("Windows folder picker failed (HRESULT 0x%08X)", code)
		}
		fyne.Do(func() { done(path, err) })
	}()
	return true
}
