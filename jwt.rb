require 'openssl'
require 'active_support'
require 'active_support/core_ext/numeric'
require 'jwt'
# Private key contents
private_pem = File.read("/home/jdyson/Downloads/example-app.2017-07-24.private-key.pem")
private_key = OpenSSL::PKey::RSA.new(private_pem)

# Generate the JWT
payload = {
  # issued at time
  iat: Time.now.to_i,
  # JWT expiration time (10 minute maximum)
  exp: 10.minutes.from_now.to_i,
  # App's GitHub identifier
  iss: 
}

print JWT.encode(payload, private_key, "RS256")

