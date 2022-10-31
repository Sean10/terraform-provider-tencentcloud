package tencentcloud

import (
	"context"
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

// go test -i; go test -test.run TestAccTencentCloudVpcBandwidthPackageResources_basic -v
func TestAccTencentCloudVpcBandwidthPackageResources_basic(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBandwidthPackageResourcesDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccVpcBandwidthPackageResources,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBandwidthPackageResourcesExists("tencentcloud_vpc_bandwidth_package_resources.bandwidth_package_resources"),
					resource.TestCheckResourceAttr("tencentcloud_vpc_bandwidth_package_resources.bandwidth_package_resources", "resource_ids", "lb-5dnrkgry"),
					resource.TestCheckResourceAttr("tencentcloud_vpc_bandwidth_package_resources.bandwidth_package_resources", "network_type", "BGP"),
					resource.TestCheckResourceAttr("tencentcloud_vpc_bandwidth_package_resources.bandwidth_package_resources", "resource_type", "LoadBalance"),
				),
			},
		},
	})
}

func testAccCheckBandwidthPackageResourcesDestroy(s *terraform.State) error {
	logId := getLogId(contextNil)
	ctx := context.WithValue(context.TODO(), logIdKey, logId)
	service := VpcService{client: testAccProvider.Meta().(*TencentCloudClient).apiV3Conn}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "tencentcloud_vpc_bandwidth_package_resources" {
			continue
		}
		idSplit := strings.Split(rs.Primary.ID, FILED_SP)
		if len(idSplit) != 3 {
			return fmt.Errorf("id is broken,%s", rs.Primary.ID)
		}
		bandwidthPackageId := idSplit[0]
		resourceId := idSplit[1]
		resourceType := idSplit[2]

		bandwidthPackageResources, err := service.DescribeVpcBandwidthPackageResources(ctx, bandwidthPackageId, resourceId, resourceType)
		if err != nil {
			log.Printf("[CRITAL]%s read VPN connection failed, reason:%s\n", logId, err.Error())
			ee, ok := err.(*errors.TencentCloudSDKError)
			if !ok {
				return err
			}
			fmt.Print(ee)
			if ee.Code == "InvalidParameterValue.BandwidthPackageNotFound" {
				return nil
			} else {
				return err
			}
		} else {
			if bandwidthPackageResources != nil {
				return fmt.Errorf("vpc bandwidthPackageResources %s still exists", rs.Primary.ID)
			}
		}
	}
	return nil
}

func testAccCheckBandwidthPackageResourcesExists(r string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		logId := getLogId(contextNil)
		ctx := context.WithValue(context.TODO(), logIdKey, logId)

		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return fmt.Errorf("resource %s is not found", r)
		}

		idSplit := strings.Split(rs.Primary.ID, FILED_SP)
		if len(idSplit) != 3 {
			return fmt.Errorf("id is broken,%s", rs.Primary.ID)
		}
		bandwidthPackageId := idSplit[0]
		resourceId := idSplit[1]
		resourceType := idSplit[2]

		service := VpcService{client: testAccProvider.Meta().(*TencentCloudClient).apiV3Conn}
		bandwidthPackageResources, err := service.DescribeVpcBandwidthPackageResources(ctx, bandwidthPackageId, resourceId, resourceType)
		if bandwidthPackageResources == nil {
			return fmt.Errorf("vpc bandwidthPackageResources %s is not found", rs.Primary.ID)
		}
		if err != nil {
			return err
		}

		return nil
	}
}

const testAccVpcBandwidthPackageResources = `
resource "tencentcloud_vpc_bandwidth_package" "bandwidth_package" {
  network_type            = "BGP"
  charge_type             = "TOP5_POSTPAID_BY_MONTH"
  bandwidth_package_name  = "iac-test-002"
  tags = {
    "createdBy" = "terraform"
  }
}

resource "tencentcloud_vpc_bandwidth_package_resources" "bandwidth_package_resources" {
  resource_ids          = "lb-5dnrkgry"
  bandwidth_package_id  = tencentcloud_vpc_bandwidth_package.bandwidth_package.id
  network_type          = "BGP"
  resource_type         = "LoadBalance"
}

`
