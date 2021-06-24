package ca

type ClientCAFlags []string

func (i *ClientCAFlags) String() string {
	return ""
}

func (i *ClientCAFlags) Set(path string) error {
	*i = append(*i, path)
	return nil
}
