# Build the WinMIPS64 reference oracle (macOS clang++ / Linux g++).
#
# The upstream pipeline.cpp, utils.cpp and utils.h are copied VERBATIM into
# build/upstream/ and compiled from there, so that their #include "stdafx.h" and
# #include "mytypes.h" resolve to the shim/ directory instead of the MFC originals
# that sit next to them in third_party/.

UPSTREAM := ../../third_party/winmips64-andoni/src
BUILD    := build
GEN      := $(BUILD)/upstream

CXX      ?= c++
CXXFLAGS ?= -O2
# -fwrapv / -fno-strict-aliasing: the simulator relies on 2's complement wrap
#   (overflow detection) and on type-punned casts of the memory arrays.
# -ftrivial-auto-var-init=zero: pipeline.cpp copies partially-initialised
#   stack structs (EX_ADD/EX_MUL "idle"); zero makes them deterministic.
AUTOINIT := $(shell echo 'int main(){}' | $(CXX) -x c++ -ftrivial-auto-var-init=zero -fsyntax-only - >/dev/null 2>&1 && echo -ftrivial-auto-var-init=zero)
ifeq ($(AUTOINIT),)
$(error $(CXX) does not support -ftrivial-auto-var-init=zero (need clang >= 16 or gcc >= 12))
endif
SIMFLAGS := -std=c++17 -fwrapv -fno-strict-aliasing $(AUTOINIT)
INCS     := -Ishim -I$(GEN) -Isrc
UPWARN   := -w
OWNWARN  := -Wall -Wno-write-strings -Wno-unused-variable -Wno-unused-but-set-variable \
            -Wno-char-subscripts -Wno-unused-value -Wno-parentheses -Wno-sign-compare -Wno-unknown-warning-option

OBJS := $(BUILD)/pipeline.o $(BUILD)/utils.o $(BUILD)/doc.o $(BUILD)/main.o

all: $(BUILD)/oracle

$(GEN)/%: $(UPSTREAM)/%
	@mkdir -p $(GEN)
	cp $< $@

$(BUILD)/pipeline.o: $(GEN)/pipeline.cpp $(GEN)/utils.h shim/stdafx.h shim/mytypes.h
	$(CXX) $(CXXFLAGS) $(SIMFLAGS) $(INCS) $(UPWARN) -c $< -o $@

$(BUILD)/utils.o: $(GEN)/utils.cpp $(GEN)/utils.h shim/stdafx.h shim/mytypes.h
	$(CXX) $(CXXFLAGS) $(SIMFLAGS) $(INCS) $(UPWARN) -c $< -o $@

$(BUILD)/doc.o: src/doc.cpp src/doc.h $(GEN)/utils.h shim/stdafx.h shim/mytypes.h
	$(CXX) $(CXXFLAGS) $(SIMFLAGS) $(INCS) $(OWNWARN) -c $< -o $@

$(BUILD)/main.o: src/main.cpp src/doc.h $(GEN)/utils.h shim/stdafx.h shim/mytypes.h
	$(CXX) $(CXXFLAGS) $(SIMFLAGS) $(INCS) $(OWNWARN) -c $< -o $@

$(BUILD)/oracle: $(OBJS)
	$(CXX) $(CXXFLAGS) $(OBJS) -o $@

# Proves that the compiled upstream files are byte-identical to third_party/.
verify-upstream: $(GEN)/pipeline.cpp $(GEN)/utils.cpp $(GEN)/utils.h
	cmp $(GEN)/pipeline.cpp $(UPSTREAM)/pipeline.cpp
	cmp $(GEN)/utils.cpp $(UPSTREAM)/utils.cpp
	cmp $(GEN)/utils.h $(UPSTREAM)/utils.h
	@echo "upstream sources are verbatim"

# Quick regression: the I/O self-test must reproduce its recorded trace.
check: $(BUILD)/oracle verify-upstream
	$(BUILD)/oracle tests/io.s --input tests/io.input --max 5000 | cmp - tests/io.default.trace
	@echo "oracle self-test OK"

clean:
	rm -rf $(BUILD)

.PHONY: all clean verify-upstream check
