module github.com/taubakabylnurlybek/ap2-generated

go 1.22

require (
	github.com/taubakabylnurlybek/ap2-generated/order v0.0.0
	github.com/taubakabylnurlybek/ap2-generated/payment v0.0.0
)

replace (
	github.com/taubakabylnurlybek/ap2-generated/order => ./order
	github.com/taubakabylnurlybek/ap2-generated/payment => ./payment
)
