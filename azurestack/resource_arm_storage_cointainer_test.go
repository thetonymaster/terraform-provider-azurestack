package azurestack

import (
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAzureStackStorageContainer_basic(t *testing.T) {
	var c storage.Container

	ri := acctest.RandInt()
	rs := strings.ToLower(acctest.RandString(11))
	config := testAccAzureStackStorageContainer_basic(ri, rs, testLocation())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureStackStorageContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureStackStorageContainerExists("azurestack_storage_container.test", &c),
				),
			},
		},
	})
}

func TestAccAzureStackStorageContainer_disappears(t *testing.T) {
	var c storage.Container

	ri := acctest.RandInt()
	rs := strings.ToLower(acctest.RandString(11))
	config := testAccAzureStackStorageContainer_basic(ri, rs, testLocation())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureStackStorageContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureStackStorageContainerExists("azurestack_storage_container.test", &c),
					testAccARMStorageContainerDisappears("azurestack_storage_container.test", &c),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccAzureStackStorageContainer_root(t *testing.T) {
	var c storage.Container

	ri := acctest.RandInt()
	rs := strings.ToLower(acctest.RandString(11))
	config := testAccAzureStackStorageContainer_root(ri, rs, testLocation())

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testCheckAzureStackStorageContainerDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testCheckAzureStackStorageContainerExists("azurestack_storage_container.test", &c),
					resource.TestCheckResourceAttr("azurestack_storage_container.test", "name", "$root"),
				),
			},
		},
	})
}

func testCheckAzureStackStorageContainerExists(name string, c *storage.Container) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		name := rs.Primary.Attributes["name"]
		storageAccountName := rs.Primary.Attributes["storage_account_name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for storage container: %s", name)
		}

		armClient := testAccProvider.Meta().(*ArmClient)
		ctx := armClient.StopContext
		blobClient, accountExists, err := armClient.getBlobStorageClientForStorageAccount(ctx, resourceGroup, storageAccountName)
		if err != nil {
			return err
		}
		if !accountExists {
			return fmt.Errorf("Bad: Storage Account %q does not exist", storageAccountName)
		}

		containers, err := blobClient.ListContainers(storage.ListContainersParameters{
			Prefix:  name,
			Timeout: 90,
		})

		if len(containers.Containers) == 0 {
			return fmt.Errorf("Bad: Storage Container %q (storage account: %q) does not exist", name, storageAccountName)
		}

		var found bool
		for _, container := range containers.Containers {
			if container.Name == name {
				found = true
				*c = container
			}
		}

		if !found {
			return fmt.Errorf("Bad: Storage Container %q (storage account: %q) does not exist", name, storageAccountName)
		}

		return nil
	}
}

func testAccARMStorageContainerDisappears(name string, c *storage.Container) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		armClient := testAccProvider.Meta().(*ArmClient)
		ctx := armClient.StopContext

		storageAccountName := rs.Primary.Attributes["storage_account_name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for storage container: %s", c.Name)
		}

		blobClient, accountExists, err := armClient.getBlobStorageClientForStorageAccount(ctx, resourceGroup, storageAccountName)
		if err != nil {
			return err
		}
		if !accountExists {
			log.Printf("[INFO]Storage Account %q doesn't exist so the container won't exist", storageAccountName)
			return nil
		}

		reference := blobClient.GetContainerReference(c.Name)
		options := &storage.DeleteContainerOptions{}
		_, err = reference.DeleteIfExists(options)
		if err != nil {
			return err
		}

		return nil
	}
}

func testCheckAzureStackStorageContainerDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "azurestack_storage_container" {
			continue
		}

		name := rs.Primary.Attributes["name"]
		storageAccountName := rs.Primary.Attributes["storage_account_name"]
		resourceGroup, hasResourceGroup := rs.Primary.Attributes["resource_group_name"]
		if !hasResourceGroup {
			return fmt.Errorf("Bad: no resource group found in state for storage container: %s", name)
		}

		armClient := testAccProvider.Meta().(*ArmClient)
		ctx := armClient.StopContext
		blobClient, accountExists, err := armClient.getBlobStorageClientForStorageAccount(ctx, resourceGroup, storageAccountName)
		if err != nil {
			//If we can't get keys then the blob can't exist
			return nil
		}
		if !accountExists {
			return nil
		}

		containers, err := blobClient.ListContainers(storage.ListContainersParameters{
			Prefix:  name,
			Timeout: 90,
		})

		if err != nil {
			return nil
		}

		var found bool
		for _, container := range containers.Containers {
			if container.Name == name {
				found = true
			}
		}

		if found {
			return fmt.Errorf("Bad: Storage Container %q (storage account: %q) still exist", name, storageAccountName)
		}
	}

	return nil
}

func TestValidateArmStorageContainerName(t *testing.T) {
	validNames := []string{
		"valid-name",
		"valid02-name",
		"$root",
	}
	for _, v := range validNames {
		_, errors := validateArmStorageContainerName(v, "name")
		if len(errors) != 0 {
			t.Fatalf("%q should be a valid Storage Container Name: %q", v, errors)
		}
	}

	invalidNames := []string{
		"InvalidName1",
		"-invalidname1",
		"invalid_name",
		"invalid!",
		"ww",
		"$notroot",
		strings.Repeat("w", 65),
	}
	for _, v := range invalidNames {
		_, errors := validateArmStorageContainerName(v, "name")
		if len(errors) == 0 {
			t.Fatalf("%q should be an invalid Storage Container Name", v)
		}
	}
}

func testAccAzureStackStorageContainer_basic(rInt int, rString string, location string) string {
	return fmt.Sprintf(`
resource "azurestack_resource_group" "test" {
    name = "acctestRG-%d"
    location = "%s"
}

resource "azurestack_storage_account" "test" {
    name                     = "acctestacc%s"
    resource_group_name      = "${azurestack_resource_group.test.name}"
    location                 = "${azurestack_resource_group.test.location}"
    account_tier             = "Standard"
    account_replication_type = "LRS"

    tags {
        environment = "staging"
    }
}

resource "azurestack_storage_container" "test" {
    name = "vhds"
    resource_group_name = "${azurestack_resource_group.test.name}"
    storage_account_name = "${azurestack_storage_account.test.name}"
    container_access_type = "private"
}
`, rInt, location, rString)
}

func testAccAzureStackStorageContainer_root(rInt int, rString string, location string) string {
	return fmt.Sprintf(`
resource "azurestack_resource_group" "test" {
    name     = "acctestRG-%d"
    location = "%s"
}

resource "azurestack_storage_account" "test" {
    name                     = "acctestacc%s"
    resource_group_name      = "${azurestack_resource_group.test.name}"
    location                 = "${azurestack_resource_group.test.location}"
    account_tier             = "Standard"
    account_replication_type = "LRS"

    tags {
        environment = "staging"
    }
}

resource "azurestack_storage_container" "test" {
    name = "$root"
    resource_group_name = "${azurestack_resource_group.test.name}"
    storage_account_name = "${azurestack_storage_account.test.name}"
    container_access_type = "private"
}
`, rInt, location, rString)
}
