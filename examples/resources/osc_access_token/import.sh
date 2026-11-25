# Import a token by its id (`osc token list` shows it). The token value cannot be
# read back, so `token` stays null; the template and the lifetime are not known
# either, so describe the token with `policy`.
terraform import osc_access_token.deployer 0b6f1c2e-5d1a-4f7e-9a1c-2f3d4e5a6b7c
