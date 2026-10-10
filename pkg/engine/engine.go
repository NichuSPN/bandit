package engine

/*
#cgo !windows LDFLAGS: -L${SRCDIR}/../../bandit_engine/target/release -lbandit_engine -ldl -lm
#cgo windows LDFLAGS: -L${SRCDIR}/../../bandit_engine/target/release -lbandit_engine -lws2_32 -luserenv -lbcrypt -lntdll
#include <stdlib.h>

extern char* rust_fast_search(const char* query, const char* dir, const char* ignores);
extern char* rust_compute_diff(const char* old_str, const char* new_str, const char* label);
extern char* rust_generate_repomap(const char* dir, size_t max_tokens);
extern void rust_free_string(char* ptr);
*/
import "C"
import (
	"unsafe"
)

func FastSearch(query, targetDir, ignores string) string {
	cQuery := C.CString(query)
	cDir := C.CString(targetDir)
	cIgnores := C.CString(ignores)
	defer C.free(unsafe.Pointer(cQuery))
	defer C.free(unsafe.Pointer(cDir))
	defer C.free(unsafe.Pointer(cIgnores))

	resPtr := C.rust_fast_search(cQuery, cDir, cIgnores)
	if resPtr == nil {
		return "Empty search query or target directory."
	}
	defer C.rust_free_string(resPtr)

	return C.GoString(resPtr)
}

func ComputeDiff(oldContent, newContent, fileLabel string) string {
	cOld := C.CString(oldContent)
	cNew := C.CString(newContent)
	cLabel := C.CString(fileLabel)
	defer C.free(unsafe.Pointer(cOld))
	defer C.free(unsafe.Pointer(cNew))
	defer C.free(unsafe.Pointer(cLabel))

	resPtr := C.rust_compute_diff(cOld, cNew, cLabel)
	if resPtr == nil {
		return ""
	}
	defer C.rust_free_string(resPtr)

	return C.GoString(resPtr)
}

func GenerateRepoMap(rootDir string, maxTokens int) string {
	cDir := C.CString(rootDir)
	defer C.free(unsafe.Pointer(cDir))

	resPtr := C.rust_generate_repomap(cDir, C.size_t(maxTokens))
	if resPtr == nil {
		return ""
	}
	defer C.rust_free_string(resPtr)

	return C.GoString(resPtr)
}
