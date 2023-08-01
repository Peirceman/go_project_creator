GO_FILES := $(shell find . -name "*.go" ! -name "*_test.go" -type f)
OUTPUT_EXE := create_go_project

ifdef OS
	OUTPUT_EXE := $(OUTPUT_EXE).exe
endif

$(OUTPUT_EXE): $(GO_FILES)
	go build -o $(OUTPUT_EXE)