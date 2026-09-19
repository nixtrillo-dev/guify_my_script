//go:build android

package main

import (
	"fmt"
	"guify_my_script/config_gen"
	"guify_my_script/endpoint_utils"
	"guify_my_script/go_gen"
	"guify_my_script/gui_gen"
	"guify_my_script/internal_utils"
	"guify_my_script/logger"
	"guify_my_script/package_installer"
	"guify_my_script/rust_gen"
	"os"
	"strconv"
)

// =====================================================================
// AUTOMATICALLY GENERATED BASH CLI ROUTER
// =====================================================================
func main() {
	if len(os.Args) < 2 {
		fmt.Println("Error: Missing command function name")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "compile_script_with_gui_config":
		if len(os.Args) < 13 {
			fmt.Println("Error: compile_script_with_gui_config requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types

		cli_inputPath := os.Args[2]

		cli_scriptType := os.Args[3]

		cli_outputJSON := os.Args[4]

		cli_excludePackages := os.Args[5]

		cli_outputBridge := os.Args[6]

		cli_binaryName := os.Args[7]

		cli_compile, err_compile := strconv.ParseBool(os.Args[8])
		if err_compile != nil {
			if os.Args[8] == "1" {
				cli_compile = true
			} else if os.Args[8] == "0" {
				cli_compile = false
			} else {
				fmt.Printf("Error: parameter 'compile' must be a valid boolean evaluation: %v\n", err_compile)
				os.Exit(1)
			}
		}

		cli_overwriteJson, err_overwriteJson := strconv.ParseBool(os.Args[9])
		if err_overwriteJson != nil {
			if os.Args[9] == "1" {
				cli_overwriteJson = true
			} else if os.Args[9] == "0" {
				cli_overwriteJson = false
			} else {
				fmt.Printf("Error: parameter 'overwriteJson' must be a valid boolean evaluation: %v\n", err_overwriteJson)
				os.Exit(1)
			}
		}

		cli_logLevel := os.Args[10]

		cli_includeFuncParams, err_includeFuncParams := strconv.ParseBool(os.Args[11])
		if err_includeFuncParams != nil {
			if os.Args[11] == "1" {
				cli_includeFuncParams = true
			} else if os.Args[11] == "0" {
				cli_includeFuncParams = false
			} else {
				fmt.Printf("Error: parameter 'includeFuncParams' must be a valid boolean evaluation: %v\n", err_includeFuncParams)
				os.Exit(1)
			}
		}

		cli_autoOpen, err_autoOpen := strconv.ParseBool(os.Args[12])
		if err_autoOpen != nil {
			if os.Args[12] == "1" {
				cli_autoOpen = true
			} else if os.Args[12] == "0" {
				cli_autoOpen = false
			} else {
				fmt.Printf("Error: parameter 'autoOpen' must be a valid boolean evaluation: %v\n", err_autoOpen)
				os.Exit(1)
			}
		}

		err := CompileScriptWithGuiConfig(cli_inputPath, cli_scriptType, cli_outputJSON, cli_excludePackages, cli_outputBridge, cli_binaryName, cli_compile, cli_overwriteJson, cli_logLevel, cli_includeFuncParams, cli_autoOpen)
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

	case "is_valid_directory":
		if len(os.Args) < 4 {
			fmt.Println("Error: is_valid_directory requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types

		cli_path := os.Args[2]

		var cli_logLevel_raw int
		raw_logLevel := os.Args[3]
		if raw_logLevel != "" && raw_logLevel != "null" && raw_logLevel != "nil" {
			parsed_logLevel, err_logLevel := strconv.Atoi(raw_logLevel)
			if err_logLevel != nil {
				fmt.Printf("Error: parameter 'logLevel' must be a valid integer: %v\n", err_logLevel)
				os.Exit(1)
			}
			cli_logLevel_raw = parsed_logLevel
		}
		cli_logLevel := logger.LogLevel(cli_logLevel_raw)

		endpoint_utils.IsValidDirectory(cli_path, cli_logLevel)

	case "get_directory_list_if_valid":
		if len(os.Args) < 4 {
			fmt.Println("Error: get_directory_list_if_valid requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types

		cli_path := os.Args[2]

		var cli_logLevel_raw int
		raw_logLevel := os.Args[3]
		if raw_logLevel != "" && raw_logLevel != "null" && raw_logLevel != "nil" {
			parsed_logLevel, err_logLevel := strconv.Atoi(raw_logLevel)
			if err_logLevel != nil {
				fmt.Printf("Error: parameter 'logLevel' must be a valid integer: %v\n", err_logLevel)
				os.Exit(1)
			}
			cli_logLevel_raw = parsed_logLevel
		}
		cli_logLevel := logger.LogLevel(cli_logLevel_raw)

		endpoint_utils.GetDirectoryListIfValid(cli_path, cli_logLevel)

	case "get_non_main_packages_json":
		if len(os.Args) < 6 {
			fmt.Println("Error: get_non_main_packages_json requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types

		cli_inputPath := os.Args[2]

		cli_scriptType := os.Args[3]

		cli_logLevel := os.Args[4]

		cli_noUpdateMode, err_noUpdateMode := strconv.ParseBool(os.Args[5])
		if err_noUpdateMode != nil {
			if os.Args[5] == "1" {
				cli_noUpdateMode = true
			} else if os.Args[5] == "0" {
				cli_noUpdateMode = false
			} else {
				fmt.Printf("Error: parameter 'noUpdateMode' must be a valid boolean evaluation: %v\n", err_noUpdateMode)
				os.Exit(1)
			}
		}

		_, err := endpoint_utils.GetNonMainPackagesJSON(cli_inputPath, cli_scriptType, cli_logLevel, cli_noUpdateMode)
		if err != nil {
			fmt.Printf("Execution Failure: %v\n", err)
			os.Exit(1)
		}

	case "get_most_represented_file_type":
		if len(os.Args) < 4 {
			fmt.Println("Error: get_most_represented_file_type requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types

		cli_path := os.Args[2]

		var cli_logLevel_raw int
		raw_logLevel := os.Args[3]
		if raw_logLevel != "" && raw_logLevel != "null" && raw_logLevel != "nil" {
			parsed_logLevel, err_logLevel := strconv.Atoi(raw_logLevel)
			if err_logLevel != nil {
				fmt.Printf("Error: parameter 'logLevel' must be a valid integer: %v\n", err_logLevel)
				os.Exit(1)
			}
			cli_logLevel_raw = parsed_logLevel
		}
		cli_logLevel := logger.LogLevel(cli_logLevel_raw)

		endpoint_utils.GetMostRepresentedFileType(cli_path, cli_logLevel)

	case "is_compilable":
		if len(os.Args) < 4 {
			fmt.Println("Error: is_compilable requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types

		cli_path := os.Args[2]

		var cli_logLevel_raw int
		raw_logLevel := os.Args[3]
		if raw_logLevel != "" && raw_logLevel != "null" && raw_logLevel != "nil" {
			parsed_logLevel, err_logLevel := strconv.Atoi(raw_logLevel)
			if err_logLevel != nil {
				fmt.Printf("Error: parameter 'logLevel' must be a valid integer: %v\n", err_logLevel)
				os.Exit(1)
			}
			cli_logLevel_raw = parsed_logLevel
		}
		cli_logLevel := logger.LogLevel(cli_logLevel_raw)

		endpoint_utils.IsCompilable(cli_path, cli_logLevel)

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
		if len(os.Args) < 7 {
			fmt.Println("Error: generate_path_from_dir requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types

		cli_dirPath := os.Args[2]

		cli_appendString := os.Args[3]

		cli_newExtension := os.Args[4]

		cli_logLevel := os.Args[5]

		cli_omitBaseName, err_omitBaseName := strconv.ParseBool(os.Args[6])
		if err_omitBaseName != nil {
			if os.Args[6] == "1" {
				cli_omitBaseName = true
			} else if os.Args[6] == "0" {
				cli_omitBaseName = false
			} else {
				fmt.Printf("Error: parameter 'omitBaseName' must be a valid boolean evaluation: %v\n", err_omitBaseName)
				os.Exit(1)
			}
		}

		_, err := endpoint_utils.GeneratePathFromDir(cli_dirPath, cli_appendString, cli_newExtension, cli_logLevel, cli_omitBaseName)
		if err != nil {
			fmt.Printf("Execution Failure: %v\n", err)
			os.Exit(1)
		}

	case "get_extension":
		if len(os.Args) < 4 {
			fmt.Println("Error: get_extension requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types

		cli_filePath := os.Args[2]

		cli_logLevel := os.Args[3]

		_, err := endpoint_utils.GetExtension(cli_filePath, cli_logLevel)
		if err != nil {
			fmt.Printf("Execution Failure: %v\n", err)
			os.Exit(1)
		}

	case "insert_parent_folder":
		if len(os.Args) < 5 {
			fmt.Println("Error: insert_parent_folder requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types

		cli_originalPath := os.Args[2]

		cli_newFolder := os.Args[3]

		cli_logLevel := os.Args[4]

		_, err := endpoint_utils.InsertParentFolder(cli_originalPath, cli_newFolder, cli_logLevel)
		if err != nil {
			fmt.Printf("Execution Failure: %v\n", err)
			os.Exit(1)
		}

	case "get_default_bridge_path_for_script":
		if len(os.Args) < 5 {
			fmt.Println("Error: get_default_bridge_path_for_script requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types

		cli_dirPath := os.Args[2]

		cli_scriptType := os.Args[3]

		cli_logLevel := os.Args[4]

		_, err := endpoint_utils.GetDefaultBridgePathForScript(cli_dirPath, cli_scriptType, cli_logLevel)
		if err != nil {
			fmt.Printf("Execution Failure: %v\n", err)
			os.Exit(1)
		}

	case "get_go_non_main_packages_json":
		if len(os.Args) < 4 {
			fmt.Println("Error: get_go_non_main_packages_json requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types

		cli_inputPath := os.Args[2]

		cli_logLevel := os.Args[3]

		_, err := go_gen.GetGoNonMainPackagesJSON(cli_inputPath, cli_logLevel)
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

		_, _, err := go_gen.GenerateBridge(cli_inputPath, cli_outputPath, cli_excludePackages, cli_logLevelStr, cli_includeFuncParams)
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

		err := go_gen.CompileLocal(cli_targetDir, cli_absBridge, cli_binaryName, cli_logLevelStr)
		if err != nil {
			fmt.Printf("Execution Failure: %v\n", err)
			os.Exit(1)
		}

	case "is_valid_execution_mode":
		if len(os.Args) < 3 {
			fmt.Println("Error: is_valid_execution_mode requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types

		cli_mode := os.Args[2]

		gui_gen.IsValidExecutionMode(cli_mode)

	case "normalize_execution_mode":
		if len(os.Args) < 3 {
			fmt.Println("Error: normalize_execution_mode requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types

		cli_mode := os.Args[2]

		gui_gen.NormalizeExecutionMode(cli_mode)

	case "normalize_execution_mode_longhand":
		if len(os.Args) < 3 {
			fmt.Println("Error: normalize_execution_mode_longhand requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types

		cli_mode := os.Args[2]

		gui_gen.NormalizeExecutionModeLonghand(cli_mode)

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

	case "resolve_binary_output_path":
		if len(os.Args) < 4 {
			fmt.Println("Error: resolve_binary_output_path requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types

		cli_name := os.Args[2]

		cli_anchorDir := os.Args[3]

		_, err := internal_utils.ResolveBinaryOutputPath(cli_name, cli_anchorDir)
		if err != nil {
			fmt.Printf("Execution Failure: %v\n", err)
			os.Exit(1)
		}

	case "parse_log_level":
		if len(os.Args) < 3 {
			fmt.Println("Error: parse_log_level requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types

		cli_level := os.Args[2]

		logger.ParseLogLevel(cli_level)

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

	case "get_rust_non_main_packages_json":
		if len(os.Args) < 5 {
			fmt.Println("Error: get_rust_non_main_packages_json requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types

		cli_inputPath := os.Args[2]

		cli_logLevel := os.Args[3]

		cli_noUpdateMode, err_noUpdateMode := strconv.ParseBool(os.Args[4])
		if err_noUpdateMode != nil {
			if os.Args[4] == "1" {
				cli_noUpdateMode = true
			} else if os.Args[4] == "0" {
				cli_noUpdateMode = false
			} else {
				fmt.Printf("Error: parameter 'noUpdateMode' must be a valid boolean evaluation: %v\n", err_noUpdateMode)
				os.Exit(1)
			}
		}

		_, err := rust_gen.GetRustNonMainPackagesJSON(cli_inputPath, cli_logLevel, cli_noUpdateMode)
		if err != nil {
			fmt.Printf("Execution Failure: %v\n", err)
			os.Exit(1)
		}

	case "generate_rust_bridge":
		if len(os.Args) < 7 {
			fmt.Println("Error: generate_rust_bridge requires parameters")
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

		_, _, err := rust_gen.GenerateRustBridge(cli_inputPath, cli_outputPath, cli_excludePackages, cli_logLevelStr, cli_includeFuncParams)
		if err != nil {
			fmt.Printf("Execution Failure: %v\n", err)
			os.Exit(1)
		}

	case "compile_rust":
		if len(os.Args) < 6 {
			fmt.Println("Error: compile_rust requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types

		cli_targetDir := os.Args[2]

		cli_absBridge := os.Args[3]

		cli_binaryName := os.Args[4]

		cli_logLevelStr := os.Args[5]

		err := rust_gen.CompileRust(cli_targetDir, cli_absBridge, cli_binaryName, cli_logLevelStr)
		if err != nil {
			fmt.Printf("Execution Failure: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Printf("Error: Unknown function command '%s'\n", os.Args[1])
		os.Exit(1)
	}
}
