package cvm

import (
	"context"
	"log"
	"sort"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	tccommon "github.com/tencentcloudstack/terraform-provider-tencentcloud/tencentcloud/common"
	"github.com/tencentcloudstack/terraform-provider-tencentcloud/tencentcloud/internal/helper"
)

func DataSourceTencentCloudInstanceTypes() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceTencentCloudInstanceTypesRead,

		Schema: map[string]*schema.Schema{
			"cpu_core_count": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The number of CPU cores of the instance.",
			},
			"gpu_core_count": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "The number of GPU cores of the instance.",
			},
			"memory_size": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Instance memory capacity, unit in GB.",
			},
			"availability_zone": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"filter"},
				Description:   "The available zone that the CVM instance locates at. This field is conflict with `filter`.",
			},
			"exclude_sold_out": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Indicate to filter instances types that is sold out or not, default is false.",
			},
			"filter": {
				Type:          schema.TypeSet,
				Optional:      true,
				MaxItems:      10,
				ConflictsWith: []string{"availability_zone"},
				Description:   "One or more name/value pairs to filter. This field is conflict with `availability_zone`.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The filter name. Valid values: `zone`, `instance-family` and `instance-charge-type`.",
						},
						"values": {
							Type:        schema.TypeList,
							Required:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Description: "The filter values.",
						},
					},
				},
			},
			"result_output_file": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Used to save results.",
			},
			"instance_types": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "An information list of cvm instance. Each element contains the following attributes:",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"availability_zone": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The available zone that the CVM instance locates at.",
						},
						"instance_type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Type of the instance.",
						},
						"cpu_core_count": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of CPU cores of the instance.",
						},
						"gpu_core_count": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of GPU cores of the instance.",
						},
						"memory_size": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Instance memory capacity, unit in GB.",
						},
						"family": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Type series of the instance.",
						},
						"instance_charge_type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Charge type of the instance.",
						},
						"status": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Sell status of the instance.",
						},
						"price": {
							Type:        schema.TypeFloat,
							Computed:    true,
							Description: "Instance price per hour.",
						},
					},
				},
			},
		},
	}
}

type instanceTypeSort struct {
	instanceTypes []*cvm.InstanceTypeQuotaItem
}

func (s instanceTypeSort) Len() int {
	return len(s.instanceTypes)
}

func (s instanceTypeSort) Swap(i, j int) {
	s.instanceTypes[i], s.instanceTypes[j] = s.instanceTypes[j], s.instanceTypes[i]
}

func (s instanceTypeSort) Less(i, j int) bool {
	// 按照价格从低到高排序
	if s.instanceTypes[i].Price != nil && s.instanceTypes[j].Price != nil {
		return *s.instanceTypes[i].Price.UnitPriceDiscount < *s.instanceTypes[j].Price.UnitPriceDiscount
	}
	return false
}

func dataSourceTencentCloudInstanceTypesRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	defer tccommon.LogElapsed("data_source.tencentcloud_instance_types.read")()
	logId := tccommon.GetLogId(ctx)
	ctx = context.WithValue(ctx, tccommon.LogIdKey, logId)

	cvmService := CvmService{
		client: meta.(tccommon.ProviderMeta).GetAPIV3Conn(),
	}

	filter := make(map[string][]string)
	if v, ok := d.GetOk("filter"); ok {
		for _, v := range v.(*schema.Set).List() {
			item := v.(map[string]interface{})
			name := item["name"].(string)
			values := helper.InterfacesStrings(item["values"].([]interface{}))
			filter[name] = values
		}
	}

	// 处理可用区参数
	if v, ok := d.GetOk("availability_zone"); ok {
		filter["zone"] = []string{v.(string)}
	}

	var instanceTypes []*cvm.InstanceTypeQuotaItem
	var errRet error

	err := resource.Retry(tccommon.ReadRetryTimeout, func() *resource.RetryError {
		instanceTypes, errRet = cvmService.DescribeInstancesSellTypeByFilter(ctx, filter)
		if errRet != nil {
			return tccommon.RetryError(errRet, tccommon.InternalError)
		}
		return nil
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// 过滤掉已售罄的实例类型
	if v, ok := d.GetOkExists("exclude_sold_out"); ok && v.(bool) {
		var available []*cvm.InstanceTypeQuotaItem
		for _, instanceType := range instanceTypes {
			if instanceType.Status != nil && *instanceType.Status == "SELL" {
				available = append(available, instanceType)
			}
		}
		instanceTypes = available
	}

	// 按照CPU、GPU和内存过滤
	if v, ok := d.GetOk("cpu_core_count"); ok {
		cpuCount := v.(int)
		var matched []*cvm.InstanceTypeQuotaItem
		for _, instanceType := range instanceTypes {
			if instanceType.Cpu != nil && int(*instanceType.Cpu) == cpuCount {
				matched = append(matched, instanceType)
			}
		}
		instanceTypes = matched
	}

	if v, ok := d.GetOk("gpu_core_count"); ok {
		gpuCount := v.(int)
		var matched []*cvm.InstanceTypeQuotaItem
		for _, instanceType := range instanceTypes {
			if instanceType.Gpu != nil && int(*instanceType.Gpu) == gpuCount {
				matched = append(matched, instanceType)
			}
		}
		instanceTypes = matched
	}

	if v, ok := d.GetOk("memory_size"); ok {
		memSize := v.(int)
		var matched []*cvm.InstanceTypeQuotaItem
		for _, instanceType := range instanceTypes {
			if instanceType.Memory != nil && int(*instanceType.Memory) == memSize {
				matched = append(matched, instanceType)
			}
		}
		instanceTypes = matched
	}

	// 按价格排序
	sort.SliceStable(instanceTypes, func(i, j int) bool {
		if instanceTypes[i].Price != nil && instanceTypes[j].Price != nil {
			return *instanceTypes[i].Price.UnitPriceDiscount < *instanceTypes[j].Price.UnitPriceDiscount
		}
		return false
	})

	instanceTypeList := make([]map[string]interface{}, 0, len(instanceTypes))
	ids := make([]string, 0, len(instanceTypes))
	for _, instanceType := range instanceTypes {
		mapping := map[string]interface{}{
			"availability_zone":    helper.PString(instanceType.Zone),
			"instance_type":        helper.PString(instanceType.InstanceType),
			"cpu_core_count":       helper.PInt64(instanceType.Cpu),
			"gpu_core_count":       helper.PInt64(instanceType.Gpu),
			"memory_size":          helper.PInt64(instanceType.Memory),
			"family":               helper.PString(instanceType.InstanceFamily),
			"instance_charge_type": helper.PString(instanceType.InstanceChargeType),
			"status":               helper.PString(instanceType.Status),
		}
		if instanceType.Price != nil && instanceType.Price.UnitPriceDiscount != nil {
			mapping["price"] = *instanceType.Price.UnitPriceDiscount
		}
		instanceTypeList = append(instanceTypeList, mapping)
		ids = append(ids, *instanceType.InstanceType)
	}

	d.SetId(helper.DataResourceIdsHash(ids))
	if e := d.Set("instance_types", instanceTypeList); e != nil {
		log.Printf("[CRITAL]%s provider set instance types list fail, reason:%s\n", logId, e.Error())
		return diag.FromErr(e)
	}

	output, ok := d.GetOk("result_output_file")
	if ok && output.(string) != "" {
		if e := tccommon.WriteToFile(output.(string), instanceTypeList); e != nil {
			return diag.FromErr(e)
		}
	}

	return nil
}
