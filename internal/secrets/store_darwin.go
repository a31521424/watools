//go:build darwin && cgo

package secrets

/*
#cgo LDFLAGS: -framework Security -framework CoreFoundation
#include <Security/Security.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>
#include <string.h>
static CFMutableDictionaryRef waQuery(const char *account) {
 CFMutableDictionaryRef q=CFDictionaryCreateMutable(NULL,0,&kCFTypeDictionaryKeyCallBacks,&kCFTypeDictionaryValueCallBacks);
 CFStringRef name=CFStringCreateWithCString(NULL,account,kCFStringEncodingUTF8);
 CFDictionarySetValue(q,kSecClass,kSecClassGenericPassword);
 CFDictionarySetValue(q,kSecAttrService,CFSTR("watools.plugin.secrets"));
 CFDictionarySetValue(q,kSecAttrAccount,name);
 CFRelease(name);
 return q;
}
static int waSecretGet(const char *account, void **out, long *length) {
 CFMutableDictionaryRef q=waQuery(account);
 CFDictionarySetValue(q,kSecReturnData,kCFBooleanTrue);
 CFDictionarySetValue(q,kSecMatchLimit,kSecMatchLimitOne);
 CFDataRef data=NULL; OSStatus result=SecItemCopyMatching(q,(CFTypeRef *)&data);CFRelease(q);
 if(result==errSecItemNotFound){*out=NULL;*length=0;return 0;}
 if(result!=errSecSuccess)return result;
 *length=CFDataGetLength(data);*out=malloc(*length?*length:1);
 if(!*out){CFRelease(data);return errSecAllocate;}
 memcpy(*out,CFDataGetBytePtr(data),*length);CFRelease(data);return 0;
}
static int waSecretSet(const char *account,const void *value,long length) {
 CFMutableDictionaryRef q=waQuery(account);
 CFDataRef data=CFDataCreate(NULL,value,length);
 CFMutableDictionaryRef update=CFDictionaryCreateMutable(NULL,0,&kCFTypeDictionaryKeyCallBacks,&kCFTypeDictionaryValueCallBacks);
 CFDictionarySetValue(update,kSecValueData,data);
 OSStatus result=SecItemUpdate(q,update);
 if(result==errSecItemNotFound){CFDictionarySetValue(q,kSecValueData,data);result=SecItemAdd(q,NULL);}
 CFRelease(update);CFRelease(data);CFRelease(q);return result;
}
static int waSecretRemove(const char *account){CFMutableDictionaryRef q=waQuery(account);OSStatus result=SecItemDelete(q);CFRelease(q);return result==errSecItemNotFound?0:result;}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

func get(account string) (string, error) {
	name := C.CString(account)
	defer C.free(unsafe.Pointer(name))
	var value unsafe.Pointer
	var size C.long
	status := C.waSecretGet(name, &value, &size)
	if status != 0 {
		return "", fmt.Errorf("Keychain read failed (%d)", status)
	}
	if value == nil {
		return "", nil
	}
	defer C.free(value)
	return string(C.GoBytes(value, C.int(size))), nil
}
func set(account, value string) error {
	name := C.CString(account)
	defer C.free(unsafe.Pointer(name))
	data := C.CBytes([]byte(value))
	defer C.free(data)
	if status := C.waSecretSet(name, data, C.long(len(value))); status != 0 {
		return fmt.Errorf("Keychain write failed (%d)", status)
	}
	return nil
}
func remove(account string) error {
	name := C.CString(account)
	defer C.free(unsafe.Pointer(name))
	if status := C.waSecretRemove(name); status != 0 {
		return fmt.Errorf("Keychain removal failed (%d)", status)
	}
	return nil
}
