**hostname=**__<regex>__::
  to define a special _hostname_ of the system: +
  Valid values for *hostname=* are extended regular expressions (RE2) that
  must match the output of *hostname* if the exit code is 0. +
  If the command exits with a non-zero code or the regular expression does not
  match, the section will be skipped. +
  {nbsp} +
  Example: +
  [sysctl:hostname=^sap-hana-db\d+$] +
  or +
  [block:hostname=dbnode.*]

**dnsdomainname=**__<regex>__::
  to define a special _DNS domain name_ of the system: +
  Valid values for *dnsdomainname=* are extended regular expressions (RE2)
  that must match the output of *dnsdomainname* if the exit code is 0. +
  If the command exits with a non-zero code (e.g. if no DNS domain name is
  configured or resolvable) or the regular expression does not match, the
  section will be skipped. +
  {nbsp} +
  Example: +
  [sysctl:dnsdomainname=^corp\.example\.com$] +
  or +
  [sys:dnsdomainname=.*\.sap\.internal$]

**fqdn=**__<regex>__::
  to define a special _fully qualified domain name (FQDN)_ of the system: +
  Valid values for *fqdn=* are extended regular expressions (RE2) that must
  match the output of *hostname -f* if the exit code is 0. +
  If the command exits with a non-zero code (e.g. if the FQDN cannot be
  resolved) or the regular expression does not match, the section will be
  skipped. +
  {nbsp} +
  Example: +
  [sysctl:fqdn=^hana\d+\.corp\.example\.com$] +
  or +
  [vm:fqdn=.*\.prod\.acme\.com$]
