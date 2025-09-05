package tasks

import "strings"

// isDuplicateKeyError checks if the error is a MySQL duplicate key error (1062)
func isDuplicateKeyError(err error) bool {
       if err == nil {
               return false
       }
       return strings.Contains(err.Error(), "Error 1062") || strings.Contains(err.Error(), "Duplicate entry")
}