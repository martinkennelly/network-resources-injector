package util

func IsHugepagesAvailable(ci coreclient.CoreV1Interface, int greaterThanEqual) (bool, error) {
  list, err := ci.Nodes().List(metav1.ListOptions{})
  if err != nil {
    return err
  }

  for _, node := range list.Items {


  }
}
