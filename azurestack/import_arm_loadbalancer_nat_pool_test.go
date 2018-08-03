package azurestack

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
)

func TestAccAzureStackLoadBalancerNatPool_importBasic(t *testing.T) {
	resourceName := "azurestack_lb_nat_pool.test"

	ri := acctest.RandInt()
	natPoolName := fmt.Sprintf("NatPool-%d", ri)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureStackLoadBalancerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAzureStackLoadBalancerNatPool_basic(ri, natPoolName, testLocation()),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// location is deprecated and was never actually used
				ImportStateVerifyIgnore: []string{"location"},
			},
		},
	})
}
