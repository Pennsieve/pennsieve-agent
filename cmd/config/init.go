package config

//InitDB is used in every (except config) CMD to initialize configuration and DB.
//func InitDB(usStore store.UserSettingsStore, uiStore store.UserInfoStore) (*sql.DB, error) {
//
//	// Read configuration variables from config.ini file.
//	if err := viper.ReadInConfig(); err != nil {
//		fmt.Println("No Pennsieve configuration file exists.")
//		fmt.Println("\nPlease use `pennsieve-agent config wizard` to setup your Pennsieve profile.")
//		os.Exit(1)
//	}
//
//	// Initialize SQLITE database
//	_, err := dbConfig.InitializeDB()
//
//	// Get current user-settings. This is either 0, or 1 entry.
//	_, err = usStore.Get()
//	if err != nil {
//		fmt.Println("Setup database")
//		migrations.Run()
//
//		selectedProfile := viper.GetString("global.default_profile")
//		fmt.Println("Selected Profile: ", selectedProfile)
//
//		if selectedProfile == "" {
//			log.Fatalf("No default profile defined in %s. Please update configuration.\n\n",
//				viper.ConfigFileUsed())
//		}
//
//		// Create new user settings
//		params := store.UserSettingsParams{
//			UserId:  "",
//			Profile: selectedProfile,
//		}
//		_, err = usStore.CreateNewUserSettings(params)
//		if err != nil {
//			log.Fatalln("Error Creating new UserSettings")
//		}
//
//	}
//
//	_, err = uiStore.UpdateActiveUser()
//	if err != nil {
//		log.Fatalln("Unable to get active user: ", err)
//	}
//}
