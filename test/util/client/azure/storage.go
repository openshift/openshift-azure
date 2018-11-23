//+build e2e

package azure

func (az *Client) ListKeys(resourceGroup string) (errs []error) {
	accts, err := az.accsc.ListByResourceGroup(az.ctx, resourceGroup)
	if err != nil {
		errs = append(errs, err)
		return
	}
	for _, acct := range *accts.Value {
		az.log.Debugf("trying to read account %s", *acct.Name)
		if acct.Tags["type"] != nil && *acct.Tags["type"] == "config" {
			// should throw an error when trying to list the keys with the given name
			_, err := az.accsc.ListKeys(az.ctx, resourceGroup, *acct.Name)
			if err != nil {
				az.log.Debugf("can't read %s, OK", *acct.Name)
				errs = append(errs, err)
			}
		}
	}
	return
}
