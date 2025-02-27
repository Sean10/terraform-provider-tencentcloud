package gaap

import (
	"context"
	"errors"
	"log"

	tccommon "github.com/tencentcloudstack/terraform-provider-tencentcloud/tencentcloud/common"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"

	"github.com/tencentcloudstack/terraform-provider-tencentcloud/tencentcloud/internal/helper"
)

func DataSourceTencentCloudGaapLayer7Listeners() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceTencentCloudGaapLayer7ListenersRead,
		Schema: map[string]*schema.Schema{
			"protocol": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: tccommon.ValidateAllowedStringValue([]string{"HTTP", "HTTPS"}),
				Description:  "Protocol of the layer7 listener to be queried. Valid values: `HTTP` and `HTTPS`.",
			},
			"proxy_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "ID of the GAAP proxy to be queried.",
			},
			"group_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Group id.",
			},
			"listener_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "ID of the layer7 listener to be queried.",
			},
			"listener_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Name of the layer7 listener to be queried.",
			},
			"port": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: tccommon.ValidatePort,
				Description:  "Port of the layer7 listener to be queried.",
			},
			"result_output_file": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Used to save results.",
			},

			// computed
			"listeners": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "An information list of layer7 listeners. Each element contains the following attributes:",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"protocol": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Protocol of the layer7 listener.",
						},
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "ID of the layer7 listener.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Name of the layer7 listener.",
						},
						"proxy_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "ID of the GAAP proxy.",
						},
						"port": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Port of the layer7 listener.",
						},
						"status": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Status of the layer7 listener.",
						},
						"certificate_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Certificate ID of the layer7 listener.",
						},
						"client_certificate_id": {
							Deprecated:  "It has been deprecated from version 1.26.0. Use `client_certificate_ids` instead.",
							Type:        schema.TypeString,
							Computed:    true,
							Description: "ID of the client certificate.",
						},
						"client_certificate_ids": {
							Type:        schema.TypeList,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Computed:    true,
							Description: "ID list of the client certificate.",
						},
						"auth_type": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Authentication type of the layer7 listener. `0` is one-way authentication and `1` is mutual authentication.",
						},
						"forward_protocol": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Protocol type of the forwarding.",
						},
						"create_time": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Creation time of the layer7 listener.",
						},
						"tls_support_versions": {
							Type:        schema.TypeSet,
							Computed:    true,
							Elem:        &schema.Schema{Type: schema.TypeString},
							Set:         schema.HashString,
							Description: "TLS version, optional TLSv1, TLSv1.1, TLSv1.2, TLSv1.3.",
						},
						"tls_ciphers": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Password Suite, optional GAAP_TLS_CIPHERS_STRICT, GAAP_TLS_CIPHERS_GENERAL, GAAP_TLS_CIPHERS_WIDE(default).",
						},
					},
				},
			},
		},
	}
}

func dataSourceTencentCloudGaapLayer7ListenersRead(d *schema.ResourceData, m interface{}) error {
	defer tccommon.LogElapsed("data_source.tencentcloud_gaap_layer7_listeners.read")()
	logId := tccommon.GetLogId(tccommon.ContextNil)
	ctx := context.WithValue(context.TODO(), tccommon.LogIdKey, logId)

	protocol := d.Get("protocol").(string)

	var (
		proxyId    *string
		groupId    *string
		listenerId *string
		name       *string
		port       *int
		ids        []string
		listeners  []map[string]interface{}
	)

	if raw, ok := d.GetOk("proxy_id"); ok {
		proxyId = helper.String(raw.(string))
	}
	if raw, ok := d.GetOk("group_id"); ok {
		groupId = helper.String(raw.(string))
	}
	if raw, ok := d.GetOk("listener_id"); ok {
		listenerId = helper.String(raw.(string))
	}

	if proxyId == nil && groupId == nil && listenerId == nil {
		return errors.New("One of proxy_id, group_id or listener_id must be set")
	}

	if raw, ok := d.GetOk("listener_name"); ok {
		name = helper.String(raw.(string))
	}
	if raw, ok := d.GetOk("port"); ok {
		port = common.IntPtr(raw.(int))
	}

	service := GaapService{client: m.(tccommon.ProviderMeta).GetAPIV3Conn()}

	switch protocol {
	case "HTTP":
		httpListeners, err := service.DescribeHTTPListeners(ctx, proxyId, groupId, listenerId, name, port)
		if err != nil {
			return err
		}

		ids = make([]string, 0, len(httpListeners))
		listeners = make([]map[string]interface{}, 0, len(httpListeners))

		for _, ls := range httpListeners {
			if ls.ListenerId == nil {
				return errors.New("listener id is nil")
			}
			if ls.ListenerName == nil {
				return errors.New("listener name is nil")
			}
			if ls.Port == nil {
				return errors.New("listener port is nil")
			}
			if ls.ListenerStatus == nil {
				return errors.New("listener status is nil")
			}
			if ls.CreateTime == nil {
				return errors.New("listener create time is nil")
			}

			ids = append(ids, *ls.ListenerId)
			m := map[string]interface{}{
				"protocol":    "HTTP",
				"id":          *ls.ListenerId,
				"name":        *ls.ListenerName,
				"port":        *ls.Port,
				"status":      *ls.ListenerStatus,
				"create_time": helper.FormatUnixTime(*ls.CreateTime),
			}

			if ls.ProxyId != nil {
				m["proxy_id"] = *ls.ProxyId
			}
			if ls.GroupId != nil {
				m["group_id"] = *ls.GroupId
			}

			listeners = append(listeners, m)

		}

	case "HTTPS":
		httpsListeners, err := service.DescribeHTTPSListeners(ctx, proxyId, groupId, listenerId, name, port)
		if err != nil {
			return err
		}

		ids = make([]string, 0, len(httpsListeners))
		listeners = make([]map[string]interface{}, 0, len(httpsListeners))

		for _, ls := range httpsListeners {
			if ls.ListenerId == nil {
				return errors.New("listener id is nil")
			}
			if ls.ListenerName == nil {
				return errors.New("listener name is nil")
			}
			if ls.Port == nil {
				return errors.New("listener port is nil")
			}
			if ls.ListenerStatus == nil {
				return errors.New("listener status is nil")
			}
			if ls.CertificateId == nil {
				return errors.New("listener certificate id is nil")
			}
			if ls.AuthType == nil {
				return errors.New("listener auth type is nil")
			}
			if ls.ForwardProtocol == nil {
				return errors.New("listener forward protocol is nil")
			}
			if ls.CreateTime == nil {
				return errors.New("listener create time is nil")
			}

			ids = append(ids, *ls.ListenerId)

			var (
				clientCertificateId      *string
				polyClientCertificateIds []*string
			)

			if *ls.AuthType == 1 {
				clientCertificateId = ls.PolyClientCertificateAliasInfo[0].CertificateId
				for _, poly := range ls.PolyClientCertificateAliasInfo {
					polyClientCertificateIds = append(polyClientCertificateIds, poly.CertificateId)
				}
			}

			m := map[string]interface{}{
				"protocol":               "HTTPS",
				"id":                     ls.ListenerId,
				"name":                   ls.ListenerName,
				"port":                   ls.Port,
				"status":                 ls.ListenerStatus,
				"certificate_id":         ls.CertificateId,
				"auth_type":              ls.AuthType,
				"forward_protocol":       ls.ForwardProtocol,
				"create_time":            helper.FormatUnixTime(*ls.CreateTime),
				"client_certificate_id":  clientCertificateId,
				"client_certificate_ids": polyClientCertificateIds,
				"tls_ciphers":            ls.TLSCiphers,
				"tls_support_versions":   helper.PStrings(ls.TLSSupportVersion),
			}
			if ls.ProxyId != nil {
				m["proxy_id"] = *ls.ProxyId
			}
			if ls.GroupId != nil {
				m["group_id"] = *ls.GroupId
			}

			listeners = append(listeners, m)
		}
	}

	_ = d.Set("listeners", listeners)
	d.SetId(helper.DataResourceIdsHash(ids))

	if output, ok := d.GetOk("result_output_file"); ok && output.(string) != "" {
		if err := tccommon.WriteToFile(output.(string), listeners); err != nil {
			log.Printf("[CRITAL]%s output file[%s] fail, reason[%v]",
				logId, output.(string), err)
			return err
		}
	}

	return nil
}
