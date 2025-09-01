package custom_validator

import "fmt"

func Custom_Validator() {
	tag := GameLibraryTagDto{
		GameLibraryTagId: "123",
		TagName:          "Shooter",
		CategoryName:     "GAME_CATEGORIES", // ❌ invalid enum
	}

	if err := validateStruct(tag); err != nil {
		fmt.Println("Validation failed:", err)
		return
	}

	fmt.Println("Validation passed ✅")
}
