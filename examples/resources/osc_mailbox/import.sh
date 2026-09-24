# A workspace has one mailbox, so any id imports it. The password is not
# readable; the next apply sets the configured one.
terraform import osc_mailbox.main mailbox
