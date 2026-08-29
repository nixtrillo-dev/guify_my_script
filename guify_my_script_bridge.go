package main

import "C"
import (
	"fmt"
	"os"
	"strconv"
	"guify_my_script/bridge_gen"
	"guify_my_script/config_gen"
	"guify_my_script/endpoint_utils"
	"guify_my_script/gui_gen"
	"guify_my_script/internal_utils"
	"guify_my_script/logger"
	"guify_my_script/package_installer"
	
)

// =====================================================================
// AUTOMATICALLY GENERATED C++ EXPORTS
// =====================================================================

//export GoGetNonMainPackagesJSON
func GoGetNonMainPackagesJSON(inputPath *C.char, logLevel *C.char) C.int {
	
	go_inputPath := C.GoString(inputPath)
	
	go_logLevel := C.GoString(logLevel)
	
	
	_, err := bridge_gen.GetNonMainPackagesJSON(go_inputPath, go_logLevel)
	if err != nil {
		fmt.Printf("Pipeline Error in GetNonMainPackagesJSON: %v\n", err)
		return 0
	}
	
	return 1
}

//export GoGenerateBridge
func GoGenerateBridge(inputPath *C.char, outputPath *C.char, excludePackages *C.char, logLevelStr *C.char, includeFuncParams C.int) C.int {
	
	go_inputPath := C.GoString(inputPath)
	
	go_outputPath := C.GoString(outputPath)
	
	go_excludePackages := C.GoString(excludePackages)
	
	go_logLevelStr := C.GoString(logLevelStr)
	
	go_includeFuncParams := includeFuncParams == 1
	
	
	_, _, err := bridge_gen.GenerateBridge(go_inputPath, go_outputPath, go_excludePackages, go_logLevelStr, go_includeFuncParams)
	if err != nil {
		fmt.Printf("Pipeline Error in GenerateBridge: %v\n", err)
		return 0
	}
	
	return 1
}

//export GoCompileLocal
func GoCompileLocal(targetDir *C.char, absBridge *C.char, binaryName *C.char, logLevelStr *C.char) C.int {
	
	go_targetDir := C.GoString(targetDir)
	
	go_absBridge := C.GoString(absBridge)
	
	go_binaryName := C.GoString(binaryName)
	
	go_logLevelStr := C.GoString(logLevelStr)
	
	
	err := bridge_gen.CompileLocal(go_targetDir, go_absBridge, go_binaryName, go_logLevelStr)
	if err != nil {
		fmt.Printf("Pipeline Error in CompileLocal: %v\n", err)
		return 0
	}
	
	return 1
}

//export GoGenerateBridgeWithGuiConfig
func GoGenerateBridgeWithGuiConfig(inputPath *C.char, outputJSON *C.char, excludePackages *C.char, outputBridge *C.char, binaryName *C.char, compile C.int, overwriteJson C.int, logLevel *C.char, includeFuncParams C.int, autoOpen C.int) C.int {
	
	go_inputPath := C.GoString(inputPath)
	
	go_outputJSON := C.GoString(outputJSON)
	
	go_excludePackages := C.GoString(excludePackages)
	
	go_outputBridge := C.GoString(outputBridge)
	
	go_binaryName := C.GoString(binaryName)
	
	go_compile := compile == 1
	
	go_overwriteJson := overwriteJson == 1
	
	go_logLevel := C.GoString(logLevel)
	
	go_includeFuncParams := includeFuncParams == 1
	
	go_autoOpen := autoOpen == 1
	
	
	err := GenerateBridgeWithGuiConfig(go_inputPath, go_outputJSON, go_excludePackages, go_outputBridge, go_binaryName, go_compile, go_overwriteJson, go_logLevel, go_includeFuncParams, go_autoOpen)
	if err != nil {
		fmt.Printf("Pipeline Error in GenerateBridgeWithGuiConfig: %v\n", err)
		return 0
	}
	
	return 1
}

//export GoGenerateGUIConfigWithPrompt
func GoGenerateGUIConfigWithPrompt(inputPath *C.char, outputPath *C.char, binaryName *C.char, logLevel *C.char, autoOpen C.int) C.int {
	
	go_inputPath := C.GoString(inputPath)
	
	go_outputPath := C.GoString(outputPath)
	
	go_binaryName := C.GoString(binaryName)
	
	go_logLevel := C.GoString(logLevel)
	
	go_autoOpen := autoOpen == 1
	
	
	err := config_gen.GenerateGUIConfigWithPrompt(go_inputPath, go_outputPath, go_binaryName, go_logLevel, go_autoOpen)
	if err != nil {
		fmt.Printf("Pipeline Error in GenerateGUIConfigWithPrompt: %v\n", err)
		return 0
	}
	
	return 1
}

//export GoGenerateGUIConfig
func GoGenerateGUIConfig(inputPath *C.char, outputPath *C.char, binaryName *C.char, overwrite C.int, logLevel *C.char, autoOpen C.int) C.int {
	
	go_inputPath := C.GoString(inputPath)
	
	go_outputPath := C.GoString(outputPath)
	
	go_binaryName := C.GoString(binaryName)
	
	go_overwrite := overwrite == 1
	
	go_logLevel := C.GoString(logLevel)
	
	go_autoOpen := autoOpen == 1
	
	
	err := config_gen.GenerateGUIConfig(go_inputPath, go_outputPath, go_binaryName, go_overwrite, go_logLevel, go_autoOpen)
	if err != nil {
		fmt.Printf("Pipeline Error in GenerateGUIConfig: %v\n", err)
		return 0
	}
	
	return 1
}

//export GoModifyPath
func GoModifyPath(filePath *C.char, appendString *C.char, newExtension *C.char, logLevel *C.char) C.int {
	
	go_filePath := C.GoString(filePath)
	
	go_appendString := C.GoString(appendString)
	
	go_newExtension := C.GoString(newExtension)
	
	go_logLevel := C.GoString(logLevel)
	
	
	_, err := endpoint_utils.ModifyPath(go_filePath, go_appendString, go_newExtension, go_logLevel)
	if err != nil {
		fmt.Printf("Pipeline Error in ModifyPath: %v\n", err)
		return 0
	}
	
	return 1
}

//export GoGeneratePathFromDir
func GoGeneratePathFromDir(dirPath *C.char, appendString *C.char, newExtension *C.char, logLevel *C.char) C.int {
	
	go_dirPath := C.GoString(dirPath)
	
	go_appendString := C.GoString(appendString)
	
	go_newExtension := C.GoString(newExtension)
	
	go_logLevel := C.GoString(logLevel)
	
	
	_, err := endpoint_utils.GeneratePathFromDir(go_dirPath, go_appendString, go_newExtension, go_logLevel)
	if err != nil {
		fmt.Printf("Pipeline Error in GeneratePathFromDir: %v\n", err)
		return 0
	}
	
	return 1
}

//export GoGenerateUI
func GoGenerateUI(configPath *C.char, compileMode *C.char, logLevelStr *C.char, clearCache C.int, targetBuilds *C.char) C.int {
	
	go_configPath := C.GoString(configPath)
	
	go_compileMode := C.GoString(compileMode)
	
	go_logLevelStr := C.GoString(logLevelStr)
	
	go_clearCache := clearCache == 1
	
	go_targetBuilds := C.GoString(targetBuilds)
	
	
	err := gui_gen.GenerateUI(go_configPath, go_compileMode, go_logLevelStr, go_clearCache, go_targetBuilds)
	if err != nil {
		fmt.Printf("Pipeline Error in GenerateUI: %v\n", err)
		return 0
	}
	
	return 1
}

//export GoSplitCamelCase
func GoSplitCamelCase(src *C.char) C.int {
	
	go_src := C.GoString(src)
	
	
	internal_utils.SplitCamelCase(go_src)
	
	return 1
}

//export GoToTitleCase
func GoToTitleCase(input *C.char) C.int {
	
	go_input := C.GoString(input)
	
	
	internal_utils.ToTitleCase(go_input)
	
	return 1
}

//export GoParseLogLevel
func GoParseLogLevel(level *C.char) C.int {
	
	go_level := C.GoString(level)
	
	
	logger.ParseLogLevel(go_level)
	
	return 1
}

//export GoDebug
func GoDebug(lvl C.int, format *C.char, args *C.char) C.int {
	
	go_lvl := logger.LogLevel(int(lvl))
	
	go_format := C.GoString(format)
	
	go_args := C.GoString(args)
	
	
	logger.Debug(go_lvl, go_format, go_args)
	
	return 1
}

//export GoInfo
func GoInfo(lvl C.int, format *C.char, args *C.char) C.int {
	
	go_lvl := logger.LogLevel(int(lvl))
	
	go_format := C.GoString(format)
	
	go_args := C.GoString(args)
	
	
	logger.Info(go_lvl, go_format, go_args)
	
	return 1
}

//export GoWarn
func GoWarn(lvl C.int, format *C.char, args *C.char) C.int {
	
	go_lvl := logger.LogLevel(int(lvl))
	
	go_format := C.GoString(format)
	
	go_args := C.GoString(args)
	
	
	logger.Warn(go_lvl, go_format, go_args)
	
	return 1
}

//export GoError
func GoError(lvl C.int, format *C.char, args *C.char) C.int {
	
	go_lvl := logger.LogLevel(int(lvl))
	
	go_format := C.GoString(format)
	
	go_args := C.GoString(args)
	
	
	logger.Error(go_lvl, go_format, go_args)
	
	return 1
}

//export GoFatal
func GoFatal(lvl C.int, format *C.char, args *C.char) C.int {
	
	go_lvl := logger.LogLevel(int(lvl))
	
	go_format := C.GoString(format)
	
	go_args := C.GoString(args)
	
	
	logger.Fatal(go_lvl, go_format, go_args)
	
	return 1
}

//export GoInstallLocalPackage
func GoInstallLocalPackage(pkgPath *C.char, updateFirst C.int, logLevel C.int) C.int {
	
	go_pkgPath := C.GoString(pkgPath)
	
	go_updateFirst := updateFirst == 1
	
	go_logLevel := logger.LogLevel(int(logLevel))
	
	
	err := package_installer.InstallLocalPackage(go_pkgPath, go_updateFirst, go_logLevel)
	if err != nil {
		fmt.Printf("Pipeline Error in InstallLocalPackage: %v\n", err)
		return 0
	}
	
	return 1
}


// =====================================================================
// AUTOMATICALLY GENERATED BASH CLI ROUTER
// =====================================================================
func main() {
	if len(os.Args) < 2 {
		fmt.Println("Error: Missing command function name")
		os.Exit(1)
	}

	switch os.Args[1] {
	    case "get_non_main_packages_json":
		if len(os.Args) < 4 {
			fmt.Println("Error: get_non_main_packages_json requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types
		
		
		cli_inputPath := os.Args[2]
		
		
		cli_logLevel := os.Args[3]
		
		
		_, err := bridge_gen.GetNonMainPackagesJSON(cli_inputPath, cli_logLevel)
		if err != nil {
			fmt.Printf("Execution Failure: %v\n", err)
			os.Exit(1)
		}
		
		
	    case "generate_bridge":
		if len(os.Args) < 7 {
			fmt.Println("Error: generate_bridge requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types
		
		
		cli_inputPath := os.Args[2]
		
		
		cli_outputPath := os.Args[3]
		
		
		cli_excludePackages := os.Args[4]
		
		
		cli_logLevelStr := os.Args[5]
		
		
		cli_includeFuncParams, err_includeFuncParams := strconv.ParseBool(os.Args[6])
		if err_includeFuncParams != nil {
			if os.Args[6] == "1" {
				cli_includeFuncParams = true
			} else if os.Args[6] == "0" {
				cli_includeFuncParams = false
			} else {
				fmt.Printf("Error: parameter 'includeFuncParams' must be a valid boolean evaluation: %v\n", err_includeFuncParams)
				os.Exit(1)
			}
		}
		
		
		_, _, err := bridge_gen.GenerateBridge(cli_inputPath, cli_outputPath, cli_excludePackages, cli_logLevelStr, cli_includeFuncParams)
		if err != nil {
			fmt.Printf("Execution Failure: %v\n", err)
			os.Exit(1)
		}
		
		
	    case "compile_local":
		if len(os.Args) < 6 {
			fmt.Println("Error: compile_local requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types
		
		
		cli_targetDir := os.Args[2]
		
		
		cli_absBridge := os.Args[3]
		
		
		cli_binaryName := os.Args[4]
		
		
		cli_logLevelStr := os.Args[5]
		
		
		err := bridge_gen.CompileLocal(cli_targetDir, cli_absBridge, cli_binaryName, cli_logLevelStr)
		if err != nil {
			fmt.Printf("Execution Failure: %v\n", err)
			os.Exit(1)
		}
		
		
	    case "generate_bridge_with_gui_config":
		if len(os.Args) < 12 {
			fmt.Println("Error: generate_bridge_with_gui_config requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types
		
		
		cli_inputPath := os.Args[2]
		
		
		cli_outputJSON := os.Args[3]
		
		
		cli_excludePackages := os.Args[4]
		
		
		cli_outputBridge := os.Args[5]
		
		
		cli_binaryName := os.Args[6]
		
		
		cli_compile, err_compile := strconv.ParseBool(os.Args[7])
		if err_compile != nil {
			if os.Args[7] == "1" {
				cli_compile = true
			} else if os.Args[7] == "0" {
				cli_compile = false
			} else {
				fmt.Printf("Error: parameter 'compile' must be a valid boolean evaluation: %v\n", err_compile)
				os.Exit(1)
			}
		}
		
		
		cli_overwriteJson, err_overwriteJson := strconv.ParseBool(os.Args[8])
		if err_overwriteJson != nil {
			if os.Args[8] == "1" {
				cli_overwriteJson = true
			} else if os.Args[8] == "0" {
				cli_overwriteJson = false
			} else {
				fmt.Printf("Error: parameter 'overwriteJson' must be a valid boolean evaluation: %v\n", err_overwriteJson)
				os.Exit(1)
			}
		}
		
		
		cli_logLevel := os.Args[9]
		
		
		cli_includeFuncParams, err_includeFuncParams := strconv.ParseBool(os.Args[10])
		if err_includeFuncParams != nil {
			if os.Args[10] == "1" {
				cli_includeFuncParams = true
			} else if os.Args[10] == "0" {
				cli_includeFuncParams = false
			} else {
				fmt.Printf("Error: parameter 'includeFuncParams' must be a valid boolean evaluation: %v\n", err_includeFuncParams)
				os.Exit(1)
			}
		}
		
		
		cli_autoOpen, err_autoOpen := strconv.ParseBool(os.Args[11])
		if err_autoOpen != nil {
			if os.Args[11] == "1" {
				cli_autoOpen = true
			} else if os.Args[11] == "0" {
				cli_autoOpen = false
			} else {
				fmt.Printf("Error: parameter 'autoOpen' must be a valid boolean evaluation: %v\n", err_autoOpen)
				os.Exit(1)
			}
		}
		
		
		err := GenerateBridgeWithGuiConfig(cli_inputPath, cli_outputJSON, cli_excludePackages, cli_outputBridge, cli_binaryName, cli_compile, cli_overwriteJson, cli_logLevel, cli_includeFuncParams, cli_autoOpen)
		if err != nil {
			fmt.Printf("Execution Failure: %v\n", err)
			os.Exit(1)
		}
		
		
	    case "generate_gui_config_with_prompt":
		if len(os.Args) < 7 {
			fmt.Println("Error: generate_gui_config_with_prompt requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types
		
		
		cli_inputPath := os.Args[2]
		
		
		cli_outputPath := os.Args[3]
		
		
		cli_binaryName := os.Args[4]
		
		
		cli_logLevel := os.Args[5]
		
		
		cli_autoOpen, err_autoOpen := strconv.ParseBool(os.Args[6])
		if err_autoOpen != nil {
			if os.Args[6] == "1" {
				cli_autoOpen = true
			} else if os.Args[6] == "0" {
				cli_autoOpen = false
			} else {
				fmt.Printf("Error: parameter 'autoOpen' must be a valid boolean evaluation: %v\n", err_autoOpen)
				os.Exit(1)
			}
		}
		
		
		err := config_gen.GenerateGUIConfigWithPrompt(cli_inputPath, cli_outputPath, cli_binaryName, cli_logLevel, cli_autoOpen)
		if err != nil {
			fmt.Printf("Execution Failure: %v\n", err)
			os.Exit(1)
		}
		
		
	    case "generate_gui_config":
		if len(os.Args) < 8 {
			fmt.Println("Error: generate_gui_config requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types
		
		
		cli_inputPath := os.Args[2]
		
		
		cli_outputPath := os.Args[3]
		
		
		cli_binaryName := os.Args[4]
		
		
		cli_overwrite, err_overwrite := strconv.ParseBool(os.Args[5])
		if err_overwrite != nil {
			if os.Args[5] == "1" {
				cli_overwrite = true
			} else if os.Args[5] == "0" {
				cli_overwrite = false
			} else {
				fmt.Printf("Error: parameter 'overwrite' must be a valid boolean evaluation: %v\n", err_overwrite)
				os.Exit(1)
			}
		}
		
		
		cli_logLevel := os.Args[6]
		
		
		cli_autoOpen, err_autoOpen := strconv.ParseBool(os.Args[7])
		if err_autoOpen != nil {
			if os.Args[7] == "1" {
				cli_autoOpen = true
			} else if os.Args[7] == "0" {
				cli_autoOpen = false
			} else {
				fmt.Printf("Error: parameter 'autoOpen' must be a valid boolean evaluation: %v\n", err_autoOpen)
				os.Exit(1)
			}
		}
		
		
		err := config_gen.GenerateGUIConfig(cli_inputPath, cli_outputPath, cli_binaryName, cli_overwrite, cli_logLevel, cli_autoOpen)
		if err != nil {
			fmt.Printf("Execution Failure: %v\n", err)
			os.Exit(1)
		}
		
		
	    case "modify_path":
		if len(os.Args) < 6 {
			fmt.Println("Error: modify_path requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types
		
		
		cli_filePath := os.Args[2]
		
		
		cli_appendString := os.Args[3]
		
		
		cli_newExtension := os.Args[4]
		
		
		cli_logLevel := os.Args[5]
		
		
		_, err := endpoint_utils.ModifyPath(cli_filePath, cli_appendString, cli_newExtension, cli_logLevel)
		if err != nil {
			fmt.Printf("Execution Failure: %v\n", err)
			os.Exit(1)
		}
		
		
	    case "generate_path_from_dir":
		if len(os.Args) < 6 {
			fmt.Println("Error: generate_path_from_dir requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types
		
		
		cli_dirPath := os.Args[2]
		
		
		cli_appendString := os.Args[3]
		
		
		cli_newExtension := os.Args[4]
		
		
		cli_logLevel := os.Args[5]
		
		
		_, err := endpoint_utils.GeneratePathFromDir(cli_dirPath, cli_appendString, cli_newExtension, cli_logLevel)
		if err != nil {
			fmt.Printf("Execution Failure: %v\n", err)
			os.Exit(1)
		}
		
		
	    case "generate_ui":
		if len(os.Args) < 7 {
			fmt.Println("Error: generate_ui requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types
		
		
		cli_configPath := os.Args[2]
		
		
		cli_compileMode := os.Args[3]
		
		
		cli_logLevelStr := os.Args[4]
		
		
		cli_clearCache, err_clearCache := strconv.ParseBool(os.Args[5])
		if err_clearCache != nil {
			if os.Args[5] == "1" {
				cli_clearCache = true
			} else if os.Args[5] == "0" {
				cli_clearCache = false
			} else {
				fmt.Printf("Error: parameter 'clearCache' must be a valid boolean evaluation: %v\n", err_clearCache)
				os.Exit(1)
			}
		}
		
		
		cli_targetBuilds := os.Args[6]
		
		
		err := gui_gen.GenerateUI(cli_configPath, cli_compileMode, cli_logLevelStr, cli_clearCache, cli_targetBuilds)
		if err != nil {
			fmt.Printf("Execution Failure: %v\n", err)
			os.Exit(1)
		}
		
		
	    case "split_camel_case":
		if len(os.Args) < 3 {
			fmt.Println("Error: split_camel_case requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types
		
		
		cli_src := os.Args[2]
		
		
		internal_utils.SplitCamelCase(cli_src)
		
		
	    case "to_title_case":
		if len(os.Args) < 3 {
			fmt.Println("Error: to_title_case requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types
		
		
		cli_input := os.Args[2]
		
		
		internal_utils.ToTitleCase(cli_input)
		
		
	    case "parse_log_level":
		if len(os.Args) < 3 {
			fmt.Println("Error: parse_log_level requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types
		
		
		cli_level := os.Args[2]
		
		
		logger.ParseLogLevel(cli_level)
		
		
	    case "debug":
		if len(os.Args) < 5 {
			fmt.Println("Error: debug requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types
		
		
		var cli_lvl_raw int
		raw_lvl := os.Args[2]
		if raw_lvl != "" && raw_lvl != "null" && raw_lvl != "nil" {
			parsed_lvl, err_lvl := strconv.Atoi(raw_lvl)
			if err_lvl != nil {
				fmt.Printf("Error: parameter 'lvl' must be a valid integer: %v\n", err_lvl)
				os.Exit(1)
			}
			cli_lvl_raw = parsed_lvl
		}
		cli_lvl := logger.LogLevel(cli_lvl_raw)
		
		
		
		cli_format := os.Args[3]
		
		
		cli_args := os.Args[4]
		
		
		logger.Debug(cli_lvl, cli_format, cli_args)
		
		
	    case "info":
		if len(os.Args) < 5 {
			fmt.Println("Error: info requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types
		
		
		var cli_lvl_raw int
		raw_lvl := os.Args[2]
		if raw_lvl != "" && raw_lvl != "null" && raw_lvl != "nil" {
			parsed_lvl, err_lvl := strconv.Atoi(raw_lvl)
			if err_lvl != nil {
				fmt.Printf("Error: parameter 'lvl' must be a valid integer: %v\n", err_lvl)
				os.Exit(1)
			}
			cli_lvl_raw = parsed_lvl
		}
		cli_lvl := logger.LogLevel(cli_lvl_raw)
		
		
		
		cli_format := os.Args[3]
		
		
		cli_args := os.Args[4]
		
		
		logger.Info(cli_lvl, cli_format, cli_args)
		
		
	    case "warn":
		if len(os.Args) < 5 {
			fmt.Println("Error: warn requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types
		
		
		var cli_lvl_raw int
		raw_lvl := os.Args[2]
		if raw_lvl != "" && raw_lvl != "null" && raw_lvl != "nil" {
			parsed_lvl, err_lvl := strconv.Atoi(raw_lvl)
			if err_lvl != nil {
				fmt.Printf("Error: parameter 'lvl' must be a valid integer: %v\n", err_lvl)
				os.Exit(1)
			}
			cli_lvl_raw = parsed_lvl
		}
		cli_lvl := logger.LogLevel(cli_lvl_raw)
		
		
		
		cli_format := os.Args[3]
		
		
		cli_args := os.Args[4]
		
		
		logger.Warn(cli_lvl, cli_format, cli_args)
		
		
	    case "error":
		if len(os.Args) < 5 {
			fmt.Println("Error: error requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types
		
		
		var cli_lvl_raw int
		raw_lvl := os.Args[2]
		if raw_lvl != "" && raw_lvl != "null" && raw_lvl != "nil" {
			parsed_lvl, err_lvl := strconv.Atoi(raw_lvl)
			if err_lvl != nil {
				fmt.Printf("Error: parameter 'lvl' must be a valid integer: %v\n", err_lvl)
				os.Exit(1)
			}
			cli_lvl_raw = parsed_lvl
		}
		cli_lvl := logger.LogLevel(cli_lvl_raw)
		
		
		
		cli_format := os.Args[3]
		
		
		cli_args := os.Args[4]
		
		
		logger.Error(cli_lvl, cli_format, cli_args)
		
		
	    case "fatal":
		if len(os.Args) < 5 {
			fmt.Println("Error: fatal requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types
		
		
		var cli_lvl_raw int
		raw_lvl := os.Args[2]
		if raw_lvl != "" && raw_lvl != "null" && raw_lvl != "nil" {
			parsed_lvl, err_lvl := strconv.Atoi(raw_lvl)
			if err_lvl != nil {
				fmt.Printf("Error: parameter 'lvl' must be a valid integer: %v\n", err_lvl)
				os.Exit(1)
			}
			cli_lvl_raw = parsed_lvl
		}
		cli_lvl := logger.LogLevel(cli_lvl_raw)
		
		
		
		cli_format := os.Args[3]
		
		
		cli_args := os.Args[4]
		
		
		logger.Fatal(cli_lvl, cli_format, cli_args)
		
		
	    case "install_local_package":
		if len(os.Args) < 5 {
			fmt.Println("Error: install_local_package requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types
		
		
		cli_pkgPath := os.Args[2]
		
		
		cli_updateFirst, err_updateFirst := strconv.ParseBool(os.Args[3])
		if err_updateFirst != nil {
			if os.Args[3] == "1" {
				cli_updateFirst = true
			} else if os.Args[3] == "0" {
				cli_updateFirst = false
			} else {
				fmt.Printf("Error: parameter 'updateFirst' must be a valid boolean evaluation: %v\n", err_updateFirst)
				os.Exit(1)
			}
		}
		
		
		var cli_logLevel_raw int
		raw_logLevel := os.Args[4]
		if raw_logLevel != "" && raw_logLevel != "null" && raw_logLevel != "nil" {
			parsed_logLevel, err_logLevel := strconv.Atoi(raw_logLevel)
			if err_logLevel != nil {
				fmt.Printf("Error: parameter 'logLevel' must be a valid integer: %v\n", err_logLevel)
				os.Exit(1)
			}
			cli_logLevel_raw = parsed_logLevel
		}
		cli_logLevel := logger.LogLevel(cli_logLevel_raw)
		
		
		
		err := package_installer.InstallLocalPackage(cli_pkgPath, cli_updateFirst, cli_logLevel)
		if err != nil {
			fmt.Printf("Execution Failure: %v\n", err)
			os.Exit(1)
		}
		
		
	    default:
		fmt.Printf("Error: Unknown function command '%s'\n", os.Args[1])
		os.Exit(1)
	}
}
