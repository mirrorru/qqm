package fixtures

import (
	"fmt"
)

func makeScanMap[E fmt.Stringer](validVals []E) map[string]E {
	result := make(map[string]E, len(validVals))
	for _, v := range validVals {
		result[v.String()] = v
	}
	return result
}

// Scan implements the Scanner interface.
func scan[E interface {
	~int32
	OrNothing() E
}](value interface{}, scanMap map[string]E) (val E, err error) {
	if value == nil {
		val = E(0)
		return
	}

	ok := true
	switch v := value.(type) {
	case string:
		val, ok = scanMap[v]
	case E:
		val = v
	case int64:
		val = E(v)
	case int32:
		val = E(v)
	case int16:
		val = E(v)
	case int8:
		val = E(v)
	case int:
		val = E(v)
	case uint64:
		val = E(v)
	case uint32:
		val = E(v)
	case uint16:
		val = E(v)
	case uint8:
		val = E(v)
	case uint:
		val = E(v)
	case []byte:
		val, ok = scanMap[string(v)]
	case *string:
		if v != nil {
			return scan[E](*v, scanMap)
		}
		ok = false
	case *E:
		if v != nil {
			return scan[E](*v, scanMap)
		}
		ok = false
	case *int64:
		if v != nil {
			return scan[E](*v, scanMap)
		}
		ok = false
	case *int32:
		if v != nil {
			return scan[E](*v, scanMap)
		}
		ok = false
	case *int16:
		if v != nil {
			return scan[E](*v, scanMap)
		}
		ok = false
	case *int8:
		if v != nil {
			return scan[E](*v, scanMap)
		}
		ok = false
	case *int:
		if v != nil {
			return scan[E](*v, scanMap)
		}
		ok = false
	case *uint64:
		if v != nil {
			return scan[E](*v, scanMap)
		}
		ok = false
	case *uint32:
		if v != nil {
			return scan[E](*v, scanMap)
		}
		ok = false
	case *uint16:
		if v != nil {
			return scan[E](*v, scanMap)
		}
		ok = false
	case *uint8:
		if v != nil {
			return scan[E](*v, scanMap)
		}
		ok = false
	case *uint:
		if v != nil {
			return scan[E](*v, scanMap)
		}
		ok = false
	case *[]byte:
		if v != nil {
			return scan[E](*v, scanMap)
		}
		ok = false
	}

	if ok {
		checkedVal := val.OrNothing()
		if checkedVal != val {
			ok = false
			val = checkedVal
		}
	}

	if !ok {
		err = fmt.Errorf("invalid type %T", value)
	}

	return
}
