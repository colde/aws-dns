# aws-dns

Simply utility to answer DNS requests on services.internal hierachy.
It will then query AWS for instances roles, and respond with the private IP of any instances that match.
Could be used for a multi-tier architecture that avoids the usage of loadbalancers.