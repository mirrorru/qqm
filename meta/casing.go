// Updated at 2026-06-29
package meta

// ToSnakeCase преобразует CamelCase в snake_case за один проход.
// Работает напрямую с байтами для ASCII-строк (Go field names).
func ToSnakeCase(s string) string {
	if s == "" {
		return ""
	}

	// Оценка: максимум _ на каждую заглавную букву, начиная со второй
	maxLen := len(s) + countUpperAfterLower(s)
	buf := make([]byte, 0, maxLen)

	for i := 0; i < len(s); i++ {
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

// countUpperAfterLower считает количество заглавных букв, после которых идёт строчная.
// Используется для предвычисления ёмкости буфера.
func countUpperAfterLower(s string) int {
	count := 0
	for i := 1; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' && s[i-1] >= 'a' && s[i-1] <= 'z' {
			count++
		}
	}
	return count
}
