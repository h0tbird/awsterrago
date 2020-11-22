package main

import (

	// stdlib
	"fmt"

	// terraform-provider-aws
	"github.com/terraform-providers/terraform-provider-aws/aws"
)

func main() {
	fmt.Println("hola")
	provider := aws.Provider()
	dataSources := provider.DataSources()
	for _, v := range dataSources {
		fmt.Println(v.Name)
	}
}
