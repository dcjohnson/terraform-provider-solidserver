package solidserver

import (
  "github.com/hashicorp/terraform/helper/schema"
  "encoding/json"
  "net/url"
  "strings"
  "fmt"
  "log"
)

func resourcednszone() *schema.Resource {
  return &schema.Resource{
    Create: resourcednszoneCreate,
    Read:   resourcednszoneRead,
    Update: resourcednszoneUpdate,
    Delete: resourcednszoneDelete,
    Exists: resourcednszoneExists,
    Importer: &schema.ResourceImporter{
        State: resourcednszoneImportState,
    },

    Schema: map[string]*schema.Schema{
      "dnsserver": &schema.Schema{
        Type:        schema.TypeString,
        Description: "The managed SMART DNS server name, or DNS server name hosting the zone.",
        Required:    true,
        ForceNew:    true,
      },
      "view": &schema.Schema{
        Type:        schema.TypeString,
        Description: "The DNS view name hosting the zone.",
        Optional:    true,
        ForceNew:    true,
        Default:     "#",
      },
      "name": &schema.Schema{
        Type:        schema.TypeString,
        Description: "The Domain Name served by the zone.",
        Required:    true,
        ForceNew:    true,
      },
      "type": &schema.Schema{
        Type:         schema.TypeString,
        Description:  "The type of the Zone to create (Supported: Master).",
        ValidateFunc: resourcednszonevalidatetype,
        Optional:     true,
        ForceNew:     true,
        Default:      "Master",
      },
      "createptr":&schema.Schema{
        Type:     schema.TypeBool,
        Description: "Automaticaly create PTR records for the Zone.",
        Optional: true,
        ForceNew: false,
        Default:  false,
      },
      "class": &schema.Schema{
        Type:     schema.TypeString,
        Description: "The class associated to the Zone.",
        Optional: true,
        ForceNew: false,
        Default:  "",
      },
      "class_parameters": &schema.Schema{
        Type:     schema.TypeMap,
        Description: "The class parameters associated to the Zone.",
        Optional: true,
        ForceNew: false,
        Default: map[string]string{},
      },
    },
  }
}

func resourcednszonevalidatetype(v interface{}, _ string) ([]string, []error) {
  switch strings.ToLower(v.(string)){
    case "master":
      return nil, nil
    default:
      return nil, []error{fmt.Errorf("Unsupported Zone type.")}
  }
}

func resourcednszoneExists(d *schema.ResourceData, meta interface{}) (bool, error) {
  s := meta.(*SOLIDserver)

  // Building parameters
  parameters := url.Values{}
  parameters.Add("dnszone_id", d.Id())

  log.Printf("[DEBUG] Checking existence of DNS Zone (oid): %s", d.Id())

  // Sending the read request
  http_resp, body, _ := s.Request("get", "rest/dns_zone_info", &parameters)

  var buf [](map[string]interface{})
  json.Unmarshal([]byte(body), &buf)

  // Checking the answer
  if ((http_resp.StatusCode == 200 || http_resp.StatusCode == 201)&& len(buf) > 0) {
    return true, nil
  }

  if (len(buf) > 0) {
    if errmsg, err_exist := buf[0]["errmsg"].(string); (err_exist) {
      // Log the error
      log.Printf("[DEBUG] SOLIDServer - Unable to find DNS Zone (oid): %s (%s)", d.Id(), errmsg)
    }
  } else {
    // Log the error
    log.Printf("[DEBUG] SOLIDServer - Unable to find DNS Zone (oid): %s", d.Id())
  }

  // Unset local ID
  d.SetId("")

  return false, nil
}

func resourcednszoneCreate(d *schema.ResourceData, meta interface{}) error {
  s := meta.(*SOLIDserver)

  // Building parameters
  parameters := url.Values{}
  parameters.Add("dns_name", d.Get("dnsserver").(string))
  if (strings.Compare(d.Get("view").(string), "#") != 0) {
    parameters.Add("dnsview_name", d.Get("view").(string))
  }
  parameters.Add("dnszone_name", d.Get("name").(string))
  parameters.Add("dnszone_type", strings.ToLower(d.Get("type").(string)))
  parameters.Add("dnszone_class_name", d.Get("class").(string))

  // New only
  parameters.Add("add_flag", "new_only")

  // Building class_parameters
  class_parameters := url.Values{}

  // Generate class parameter for createptr if required
  if (d.Get("createptr").(bool)) {
    class_parameters.Add("dnsptr", "1")
  } else {
    class_parameters.Add("dnsptr", "0")
  }

  for k, v := range d.Get("class_parameters").(map[string]interface{}) {
    class_parameters.Add(k, v.(string))
  }

  parameters.Add("dnszone_class_parameters", class_parameters.Encode())

  // Sending the creation request
  http_resp, body, _ := s.Request("post", "rest/dns_zone_add", &parameters)

  var buf [](map[string]interface{})
  json.Unmarshal([]byte(body), &buf)

  // Checking the answer
  if ((http_resp.StatusCode == 200 || http_resp.StatusCode == 201)&& len(buf) > 0) {
    if oid, oid_exist := buf[0]["ret_oid"].(string); (oid_exist) {
      log.Printf("[DEBUG] SOLIDServer - Created DNS Zone (oid): %s", oid)
      d.SetId(oid)
      return nil
    }
  }

  // Reporting a failure
  return fmt.Errorf("SOLIDServer - Unable to create DNS Zone: %s", d.Get("name").(string))
}

func resourcednszoneUpdate(d *schema.ResourceData, meta interface{}) error {
  s := meta.(*SOLIDserver)

  // Building parameters
  parameters := url.Values{}
  parameters.Add("dnszone_id", d.Id())
  parameters.Add("dnszone_class_name", d.Get("class").(string))

  // Edit only
  parameters.Add("add_flag", "edit_only")

  // Building class_parameters
  class_parameters := url.Values{}

  // Generate class parameter for createptr if required
  if (d.Get("createptr").(bool)) {
    class_parameters.Add("dnsptr", "1")
  } else {
    class_parameters.Add("dnsptr", "0")
  }

  for k, v := range d.Get("class_parameters").(map[string]interface{}) {
    class_parameters.Add(k, v.(string))
  }

  parameters.Add("dnszone_class_parameters", class_parameters.Encode())

  // Sending the update request
  http_resp, body, _ := s.Request("put", "rest/dns_zone_add", &parameters)

  var buf [](map[string]interface{})
  json.Unmarshal([]byte(body), &buf)

  // Checking the answer
  if ((http_resp.StatusCode == 200 || http_resp.StatusCode == 201)&& len(buf) > 0) {
    if oid, oid_exist := buf[0]["ret_oid"].(string); (oid_exist) {
      log.Printf("[DEBUG] SOLIDServer - Updated DNS Zone (oid): %s", oid)
      d.SetId(oid)
      return nil
    }
  }

  // Reporting a failure
  return fmt.Errorf("SOLIDServer - Unable to update Zone: %s", d.Get("name").(string))
}

func resourcednszoneDelete(d *schema.ResourceData, meta interface{}) error {
  s := meta.(*SOLIDserver)

  // Building parameters
  parameters := url.Values{}
  parameters.Add("dnszone_id", d.Id())

  // Sending the deletion request
  http_resp, body, _ := s.Request("delete", "rest/dns_zone_delete", &parameters)

  var buf [](map[string]interface{})
  json.Unmarshal([]byte(body), &buf)

  // Checking the answer
  if (http_resp.StatusCode != 204 && len(buf) > 0) {
    if errmsg, err_exist := buf[0]["errmsg"].(string); (err_exist) {
      log.Printf("[DEBUG] SOLIDServer - Unable to delete Zone: %s (%s)", d.Get("name"), errmsg)
    }
  }

  // Log deletion
  log.Printf("[DEBUG] SOLIDServer - Deleted Zone (oid): %s", d.Id())

  // Unset local ID
  d.SetId("")

  return nil
}

func resourcednszoneRead(d *schema.ResourceData, meta interface{}) error {
  s := meta.(*SOLIDserver)

  // Building parameters
  parameters := url.Values{}
  parameters.Add("dnszone_id", d.Id())

  // Sending the read request
  http_resp, body, _ := s.Request("get", "rest/dns_zone_info", &parameters)

  var buf [](map[string]interface{})
  json.Unmarshal([]byte(body), &buf)

  // Checking the answer
  if (http_resp.StatusCode == 200 && len(buf) > 0) {
    d.Set("dnsserver", buf[0]["dns_name"].(string))
    d.Set("view", buf[0]["dnsview_name"].(string))
    d.Set("name", buf[0]["dnszone_name"].(string))
    d.Set("type", buf[0]["dnszone_type"].(string))

    // Updating local class_parameters
    current_class_parameters := d.Get("class_parameters").(map[string]interface{})
    retrieved_class_parameters, _ := url.ParseQuery(buf[0]["dnszone_class_parameters"].(string))
    computed_class_parameters := map[string]string{}

    if createptr, createptr_exist := retrieved_class_parameters["dnsptr"]; (createptr_exist) {
      if (createptr[0] == "1") {
        d.Set("createptr", true)
      } else {
        d.Set("createptr", false)
      }
    }

    for ck, _ := range current_class_parameters {
      if rv, rv_exist := retrieved_class_parameters[ck]; (rv_exist) {
        computed_class_parameters[ck] = rv[0]
      } else {
        computed_class_parameters[ck] = ""
      }
    }

    d.Set("class_parameters", computed_class_parameters)

    return nil
  }

  if (len(buf) > 0) {
    if errmsg, err_exist := buf[0]["errmsg"].(string); (err_exist) {
      // Log the error
      log.Printf("[DEBUG] SOLIDServer - Unable to find DNS Zone: %s (%s)", d.Get("name"), errmsg)
    }
  } else {
    // Log the error
    log.Printf("[DEBUG] SOLIDServer - Unable to find DNS Zone (oid): %s", d.Id())
  }

  // Do not unset the local ID to avoid inconsistency

  // Reporting a failure
  return fmt.Errorf("SOLIDServer - Unable to find DNS Zone: %s", d.Get("name").(string))
}

func resourcednszoneImportState(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
  s := meta.(*SOLIDserver)

  // Building parameters
  parameters := url.Values{}
  parameters.Add("dnszone_id", d.Id())

  // Sending the read request
  http_resp, body, _ := s.Request("get", "rest/dns_zone_info", &parameters)

  var buf [](map[string]interface{})
  json.Unmarshal([]byte(body), &buf)

  // Checking the answer
  if (http_resp.StatusCode == 200 && len(buf) > 0) {
    d.Set("dnsserver", buf[0]["dns_name"].(string))
    d.Set("view", buf[0]["dnsview_name"].(string))
    d.Set("name", buf[0]["dnszone_name"].(string))
    d.Set("type", buf[0]["dnszone_type"].(string))

    return []*schema.ResourceData{d}, nil
  }

  if (len(buf) > 0) {
    if errmsg, err_exist := buf[0]["errmsg"].(string); (err_exist) {
      // Log the error
      log.Printf("[DEBUG] SOLIDServer - Unable to import DNS Zone (oid): %s (%s)", d.Id(), errmsg)
    }
  } else {
    // Log the error
    log.Printf("[DEBUG] SOLIDServer - Unable to find and import DNS Zone (oid): %s", d.Id())
  }

  // Reporting a failure
  return nil, fmt.Errorf("SOLIDServer - Unable to find and import DNS Zone (oid): %s", d.Id())
}

