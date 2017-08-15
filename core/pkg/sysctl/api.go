package sysctl

// BulkModify changes the settings according to the given sysctlAdjustments,
// returns the original value and error
func BulkModify(sysctlAdjustments map[string]string) (originalSysctl map[string]string, err error) {
	originalSysctl = make(map[string]string)
	sys := New()
	for k, v := range sysctlAdjustments {
		defVar, err := sys.GetSysctl(k)
		if err != nil {
			return originalSysctl, err
		}
		originalSysctl[k] = defVar

		if err := sys.SetSysctl(k, v); err != nil {
			return originalSysctl, err
		}
	}
	return originalSysctl, nil
}
