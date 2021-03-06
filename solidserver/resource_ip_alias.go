package solidserver

import (
  "github.com/hashicorp/terraform/helper/schema"
  "encoding/json"
  "net/url"
  "strings"
  "fmt"
  "log"
)

func resourceipalias() *schema.Resource {
  return &schema.Resource{
    Create: resourceipaliasCreate,
    Read:   resourceipaliasRead,
    //Update: resourceipaliasUpdate,
    Delete: resourceipaliasDelete,

    Schema: map[string]*schema.Schema{
      "space": &schema.Schema{
        Type:     schema.TypeString,
        Description: "The name of the space to which the address belong to.",
        Required: true,
        ForceNew: true,
      },
      "address": &schema.Schema{
        Type:     schema.TypeString,
        Description: "The IP address for which the alias will be associated to.",
        Required: true,
        ForceNew: true,
      },
      "name": &schema.Schema{
        Type:     schema.TypeString,
        Description: "The FQDN of the IP address alias to create.",
        Required: true,
        ForceNew: true,
      },
      "type": &schema.Schema{
        Type:         schema.TypeString,
        Description:  "The type of the Alias to create (Supported: A, CNAME; Default: CNAME).",
        ValidateFunc: resourcealiasvalidatetype,
        Default:      "CNAME",
        Optional:     true,
        ForceNew:     true,
      },
    },
  }
}

func resourcealiasvalidatetype(v interface{}, _ string) ([]string, []error) {
  switch strings.ToUpper(v.(string)){
    case "A":
      return nil, nil
    case "CNAME":
      return nil, nil
    default:
      return nil, []error{fmt.Errorf("Unsupported Alias type.")}
  }
}

func resourceipaliasCreate(d *schema.ResourceData, meta interface{}) error {
  s := meta.(*SOLIDserver)

  var site_id    string = ipsiteidbyname(d.Get("space").(string), meta)
  var address_id  string = ipaddressidbyip(site_id, d.Get("address").(string), meta)

  // Building parameters
  parameters := url.Values{}
  parameters.Add("ip_id", address_id)
  parameters.Add("ip_name", d.Get("name").(string))
  parameters.Add("ip_name_type", d.Get("type").(string))

  // Sending the creation request
  http_resp, body, _ := s.Request("post", "rest/ip_alias_add", &parameters)

  var buf [](map[string]interface{})
  json.Unmarshal([]byte(body), &buf)

  // Checking the answer
  if ((http_resp.StatusCode == 200 || http_resp.StatusCode == 201)&& len(buf) > 0) {
    if oid, oid_exist := buf[0]["ret_oid"].(string); (oid_exist) {
      //ip_name_id := ipaliasidbyinfo(oid, d.Get("name").(string), d.Get("type").(string), meta)

      log.Printf("[DEBUG] SOLIDServer - Created IP Alias (oid): %s", oid)

      d.SetId(oid)

      return nil
    }
  }

  // Reporting a failure
  return fmt.Errorf("SOLIDServer - Unable to create IP Alias: %s - %s (associated to IP address with ID: %s)", d.Get("name").(string), d.Get("type"), address_id)
}

//func resourceipaliasUpdate(d *schema.ResourceData, meta interface{}) error {
  // Not necessary
//  return nil
//}

func resourceipaliasDelete(d *schema.ResourceData, meta interface{}) error {
  s := meta.(*SOLIDserver)

  // Building parameters
  parameters := url.Values{}
  parameters.Add("ip_name_id", d.Id())

  // Sending the deletion request
  http_resp, body, _ := s.Request("delete", "rest/ip_alias_delete", &parameters)

  var buf [](map[string]interface{})
  json.Unmarshal([]byte(body), &buf)

  // Checking the answer
  if (http_resp.StatusCode != 204 && len(buf) > 0) {
    if errmsg, err_exist := buf[0]["errmsg"].(string); (err_exist) {
      log.Printf("[DEBUG] SOLIDServer - Unable to delete IP Alias : %s - %s (%s)", d.Get("name"), d.Get("type"), errmsg)
    }
  }

  // Log deletion
  log.Printf("[DEBUG] SOLIDServer - Deleted IP Alias with oid: %s", d.Id())

  // Unset local ID
  d.SetId("")

  return nil
}

func resourceipaliasRead(d *schema.ResourceData, meta interface{}) error {
  s := meta.(*SOLIDserver)

  var site_id    string = ipsiteidbyname(d.Get("space").(string), meta)
  var address_id  string = ipaddressidbyip(site_id, d.Get("address").(string), meta)

  // Building parameters
  parameters := url.Values{}
  parameters.Add("ip_id", address_id)
  // Bug - Ticket 18653
  //parameters.Add("WHERE", "ip_name_id='" + d.Id() + "'")

  // Sending the read request
  http_resp, body, _ := s.Request("get", "rest/ip_alias_list", &parameters)

  var buf [](map[string]interface{})
  json.Unmarshal([]byte(body), &buf)

  // Shall be removed once Ticket 18653 is closed
  // Checking the answer
  if (http_resp.StatusCode == 200 && len(buf) > 0) {
    for i := 0; i < len(buf); i++ {
      r_ip_name_id, r_ip_name_id_exist := buf[i]["ip_name_id"].(string)
      r_ip_name_type, r_ip_name_type_exist := buf[i]["ip_name_type"].(string)
      r_alias_name, r_alias_name_exist := buf[i]["alias_name"].(string)

      if (r_ip_name_id_exist && strings.Compare(d.Id(), r_ip_name_id) == 0) {
        if (r_alias_name_exist) {
          d.Set("name", r_alias_name)
        }
        if (r_ip_name_type_exist) {
          d.Set("type", r_ip_name_type)
        }

        return nil
      }
    }
  }

  // Shall be restored once Ticket 18653 is closed
  // Checking the answer
  //if (http_resp.StatusCode == 200 && len(buf) > 0) {
  //  d.Set("name", buf[0]["alias_name"].(string))
  //  d.Set("type", buf[0]["ip_name_type"].(string))
  //
  //  return nil
  //}

  if (len(buf) > 0) {
    if errmsg, err_exist := buf[0]["errmsg"].(string); (err_exist) {
      // Log the error
      log.Printf("[DEBUG] SOLIDServer - Unable to find IP Alias: %s (%s)", d.Get("name"), errmsg)
    }
  } else {
    // Log the error
    log.Printf("[DEBUG] SOLIDServer - Unable to find IP Alias (oid): %s", d.Id())
  }

  // Do not unset the local ID to avoid inconsistency

  // Reporting a failure
  return fmt.Errorf("SOLIDServer - Unable to find IP Alias: %s", d.Get("name").(string))
}
