package utils

// converts singular role name to plural group
// ie coach -> coaches, student -> students, or owner -> owners
func PluralGroupNameFromRole(role string) string {
	switch role {
	case "owner":
		return "owners"
	case "coach":
		return "coaches"

	case "student":
		return "students"
	default:
		return ""
	}
}
