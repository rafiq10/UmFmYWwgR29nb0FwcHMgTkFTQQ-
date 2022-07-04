package env

import (
	"os"
)

func GetEnvVar(key, defaultV string) string {
	// wd, err := os.Getwd()
	// if err != nil {
	// 	panic(err)
	// }
	// // parent := filepath.Dir(wd)
	// fmt.Println(wd)
	// err = godotenv.Load(filepath.Join(wd, ".env"))

	// if err != nil {
	// 	log.Fatalf("Error loading .env file")
	// }
	val := os.Getenv(key)
	if val == "" {
		val = defaultV
	}
	return val
}
