# Every instance of every subscribed service in the workspace.
list "osc_instance" "all" {
  provider = osc
}

# Only the instances of one service.
list "osc_instance" "caches" {
  provider = osc
  config {
    service_id = "valkey-io-valkey"
  }
}
