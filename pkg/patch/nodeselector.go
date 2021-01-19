package patch

func CreateNodeSelector(patch []JSONOperation, existing map[string]string, desired map[string]string) []JSONOperation {
	targetMap := make(map[string]string)

	for k, v := range existing {
		targetMap[k] = v
	}

	for k, v := range desired {
		targetMap[k] = v
	}

	if len(targetMap) == 0 {
		return patch
	}
	patch = append(patch, JSONOperation{
		Operation: "add",
		Path:      "/spec/nodeSelector",
		Value:     targetMap,
	})
	return patch
}
