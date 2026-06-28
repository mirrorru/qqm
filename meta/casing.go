package meta

// ToSnakeCase преобразует CamelCase в snake_case за один проход по байтам.
// EN: ToSnakeCase converts CamelCase to snake_case in a single byte pass.
//
//nolint:gocognit, nestif
func ToSnakeCase(s string) string {
	if s == "" {
		return ""
	}

	// Оценка: максимум один _ на каждую заглавную, начиная со второй.
	// EN: Estimate: at most one _ per uppercase letter, starting from second.
	maxLen := len(s) + countUpperAfterLower(s)
	buf := make([]byte, 0, maxLen)

	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			if i > 0 {
				prev := s[i-1]
				if prev >= 'a' && prev <= 'z' {
					buf = append(buf, '_')
				} else if i+1 < len(s) {
					next := s[i+1]
					if next >= 'a' && next <= 'z' && i > 1 {
						buf = append(buf, '_')
					}
				}
			}
			buf = append(buf, c+32) // to lower
		} else {
			buf = append(buf, c)
		}
	}

	return string(buf)
}

// countUpperAfterLower считает заглавные буквы, после которых идёт строчная.
// EN: countUpperAfterLower counts uppercase letters followed by a lowercase letter.
func countUpperAfterLower(s string) int {
	count := 0
	for i := 1; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' && s[i-1] >= 'a' && s[i-1] <= 'z' {
			count++
		}
	}
	return count
}
