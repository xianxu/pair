//go:build darwin && cgo

package sessioninventory

/*
#cgo LDFLAGS: -framework CoreFoundation
#include <CoreFoundation/CoreFoundation.h>
static int codex_has_forced_preferences(void) {
 CFStringRef d=CFSTR("com.openai.codex");
 return CFPreferencesAppValueIsForced(CFSTR("config_toml_base64"),d) ||
        CFPreferencesAppValueIsForced(CFSTR("requirements_toml_base64"),d);
}
*/
import "C"

func codexForcedPreferences() (present, known bool) {
	return C.codex_has_forced_preferences() != 0, true
}
