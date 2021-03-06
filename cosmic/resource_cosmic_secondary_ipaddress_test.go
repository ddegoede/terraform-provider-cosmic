package cosmic

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	"github.com/xanzy/go-cosmic/cosmic"
)

func TestAccCosmicSecondaryIPAddress_basic(t *testing.T) {
	var ip cosmic.AddIpToNicResponse

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCosmicSecondaryIPAddressDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCosmicSecondaryIPAddress_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCosmicSecondaryIPAddressExists(
						"cosmic_secondary_ipaddress.foo", &ip),
				),
			},
		},
	})
}

func TestAccCosmicSecondaryIPAddress_fixedIP(t *testing.T) {
	var ip cosmic.AddIpToNicResponse

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckCosmicSecondaryIPAddressDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccCosmicSecondaryIPAddress_fixedIP,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCosmicSecondaryIPAddressExists(
						"cosmic_secondary_ipaddress.foo", &ip),
					testAccCheckCosmicSecondaryIPAddressAttributes(&ip),
					resource.TestCheckResourceAttr(
						"cosmic_secondary_ipaddress.foo", "ip_address", CLOUDSTACK_NETWORK_1_IPADDRESS1),
				),
			},
		},
	})
}

func testAccCheckCosmicSecondaryIPAddressExists(
	n string, ip *cosmic.AddIpToNicResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No IP address ID is set")
		}

		cs := testAccProvider.Meta().(*cosmic.CosmicClient)

		virtualmachine, ok := rs.Primary.Attributes["virtual_machine_id"]
		if !ok {
			virtualmachine, ok = rs.Primary.Attributes["virtual_machine"]
		}

		// Retrieve the virtual_machine ID
		virtualmachineid, e := retrieveID(cs, "virtual_machine", virtualmachine)
		if e != nil {
			return e.Error()
		}

		// Get the virtual machine details
		vm, count, err := cs.VirtualMachine.GetVirtualMachineByID(virtualmachineid)
		if err != nil {
			if count == 0 {
				return fmt.Errorf("Instance not found")
			}
			return err
		}

		nicid, ok := rs.Primary.Attributes["nic_id"]
		if !ok {
			nicid, ok = rs.Primary.Attributes["nicid"]
		}
		if !ok {
			nicid = vm.Nic[0].Id
		}

		p := cs.Nic.NewListNicsParams(virtualmachineid)
		p.SetNicid(nicid)

		l, err := cs.Nic.ListNics(p)
		if err != nil {
			return err
		}

		if l.Count == 0 {
			return fmt.Errorf("NIC not found")
		}

		if l.Count > 1 {
			return fmt.Errorf("Found more then one possible result: %v", l.Nics)
		}

		for _, sip := range l.Nics[0].Secondaryip {
			if sip.Id == rs.Primary.ID {
				ip.Ipaddress = sip.Ipaddress
				ip.Nicid = l.Nics[0].Id
				return nil
			}
		}

		return fmt.Errorf("IP address not found")
	}
}

func testAccCheckCosmicSecondaryIPAddressAttributes(
	ip *cosmic.AddIpToNicResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {

		if ip.Ipaddress != CLOUDSTACK_NETWORK_1_IPADDRESS1 {
			return fmt.Errorf("Bad IP address: %s", ip.Ipaddress)
		}
		return nil
	}
}

func testAccCheckCosmicSecondaryIPAddressDestroy(s *terraform.State) error {
	cs := testAccProvider.Meta().(*cosmic.CosmicClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "cosmic_secondary_ipaddress" {
			continue
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No IP address ID is set")
		}

		virtualmachine, ok := rs.Primary.Attributes["virtual_machine_id"]
		if !ok {
			virtualmachine, ok = rs.Primary.Attributes["virtual_machine"]
		}

		// Retrieve the virtual_machine ID
		virtualmachineid, e := retrieveID(cs, "virtual_machine", virtualmachine)
		if e != nil {
			return e.Error()
		}

		// Get the virtual machine details
		vm, count, err := cs.VirtualMachine.GetVirtualMachineByID(virtualmachineid)
		if err != nil {
			if count == 0 {
				return nil
			}
			return err
		}

		nicid, ok := rs.Primary.Attributes["nic_id"]
		if !ok {
			nicid, ok = rs.Primary.Attributes["nicid"]
		}
		if !ok {
			nicid = vm.Nic[0].Id
		}

		p := cs.Nic.NewListNicsParams(virtualmachineid)
		p.SetNicid(nicid)

		l, err := cs.Nic.ListNics(p)
		if err != nil {
			return err
		}

		if l.Count == 0 {
			return fmt.Errorf("NIC not found")
		}

		if l.Count > 1 {
			return fmt.Errorf("Found more then one possible result: %v", l.Nics)
		}

		for _, sip := range l.Nics[0].Secondaryip {
			if sip.Id == rs.Primary.ID {
				return fmt.Errorf("IP address %s still exists", rs.Primary.ID)
			}
		}

		return nil
	}

	return nil
}

var testAccCosmicSecondaryIPAddress_basic = fmt.Sprintf(`
resource "cosmic_instance" "foobar" {
  name = "terraform-test"
  service_offering= "%s"
  network_id = "%s"
  template = "%s"
  zone = "%s"
  expunge = true
}

resource "cosmic_secondary_ipaddress" "foo" {
	virtual_machine_id = "${cosmic_instance.foobar.id}"
}
`,
	CLOUDSTACK_SERVICE_OFFERING_1,
	CLOUDSTACK_NETWORK_1,
	CLOUDSTACK_TEMPLATE,
	CLOUDSTACK_ZONE)

var testAccCosmicSecondaryIPAddress_fixedIP = fmt.Sprintf(`
resource "cosmic_instance" "foobar" {
  name = "terraform-test"
  service_offering= "%s"
  network_id = "%s"
  template = "%s"
  zone = "%s"
  expunge = true
}

resource "cosmic_secondary_ipaddress" "foo" {
	ip_address = "%s"
	virtual_machine_id = "${cosmic_instance.foobar.id}"
}`,
	CLOUDSTACK_SERVICE_OFFERING_1,
	CLOUDSTACK_NETWORK_1,
	CLOUDSTACK_TEMPLATE,
	CLOUDSTACK_ZONE,
	CLOUDSTACK_NETWORK_1_IPADDRESS1)
