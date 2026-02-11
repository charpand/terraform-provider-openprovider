// Package ssl provides functionality for working with SSL/TLS certificates.
package ssl

// SSLOrder represents an SSL certificate order.
type SSLOrder struct {
	ID                       int    `json:"id"`
	ProductID                int    `json:"product_id"`
	CommonName               string `json:"common_name"`
	BrandName                string `json:"brand_name,omitempty"`
	Status                   string `json:"status"`
	OrderDate                string `json:"order_date"`
	ActiveDate               string `json:"active_date,omitempty"`
	ExpirationDate           string `json:"expiration_date,omitempty"`
	Autorenew                string `json:"autorenew,omitempty"`
	OwnerHandle              string `json:"owner_handle,omitempty"`
	AdminHandle              string `json:"admin_handle,omitempty"`
	BillingHandle            string `json:"billing_handle,omitempty"`
	TechnicalHandle          string `json:"technical_handle,omitempty"`
	AdditionalDomains        []string `json:"additional_domains,omitempty"`
	Certificate              string `json:"certificate,omitempty"`
	CertificateCA            string `json:"certificate_ca,omitempty"`
	DomainValidationMethod   string `json:"domain_validation_method,omitempty"`
	ApprovedBy               string `json:"approved_by,omitempty"`
	ApprovedDate             string `json:"approved_date,omitempty"`
}

// SSLProduct represents an available SSL product.
type SSLProduct struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	BrandName       string `json:"brand_name"`
	Category        string `json:"category"`
	Description     string `json:"description,omitempty"`
	DeliveryTime    string `json:"delivery_time,omitempty"`
	Encryption      string `json:"encryption,omitempty"`
	FreeRefundDays  int    `json:"free_refund_period,omitempty"`
	FreeReissueDays int    `json:"free_reissue_period,omitempty"`
}

// ListSSLOrdersResponse represents the API response for listing SSL orders.
type ListSSLOrdersResponse struct {
	Code int                         `json:"code"`
	Data ListSSLOrdersResponseData   `json:"data"`
	Desc string                      `json:"desc"`
}

// ListSSLOrdersResponseData contains the orders list data.
type ListSSLOrdersResponseData struct {
	Results []SSLOrder `json:"results"`
	Total   int        `json:"total"`
}

// GetSSLOrderResponse represents the API response for getting an SSL order.
type GetSSLOrderResponse struct {
	Code int      `json:"code"`
	Data SSLOrder `json:"data"`
}

// CreateSSLOrderRequest represents a request to create an SSL order.
type CreateSSLOrderRequest struct {
	ProductID             int      `json:"product_id"`
	CommonName            string   `json:"common_name"`
	AdditionalDomains     []string `json:"additional_domains,omitempty"`
	OwnerHandle           string   `json:"owner_handle,omitempty"`
	AdminHandle           string   `json:"admin_handle,omitempty"`
	BillingHandle         string   `json:"billing_handle,omitempty"`
	TechnicalHandle       string   `json:"technical_handle,omitempty"`
	DomainValidationMethod string  `json:"domain_validation_method,omitempty"`
	Autorenew             string   `json:"autorenew,omitempty"`
}

// CreateSSLOrderResponse represents the API response for creating an SSL order.
type CreateSSLOrderResponse struct {
	Code int      `json:"code"`
	Data SSLOrder `json:"data"`
}

// UpdateSSLOrderRequest represents a request to update an SSL order.
type UpdateSSLOrderRequest struct {
	Autorenew string `json:"autorenew,omitempty"`
}

// UpdateSSLOrderResponse represents the API response for updating an SSL order.
type UpdateSSLOrderResponse struct {
	Code int      `json:"code"`
	Data SSLOrder `json:"data"`
}

// RenewSSLOrderRequest represents a request to renew an SSL order.
type RenewSSLOrderRequest struct {
	Period int `json:"period,omitempty"`
}

// RenewSSLOrderResponse represents the API response for renewing an SSL order.
type RenewSSLOrderResponse struct {
	Code int      `json:"code"`
	Data SSLOrder `json:"data"`
}

// ReissueSSLOrderRequest represents a request to reissue an SSL order.
type ReissueSSLOrderRequest struct {
	CommonName            string   `json:"common_name,omitempty"`
	AdditionalDomains     []string `json:"additional_domains,omitempty"`
	DomainValidationMethod string  `json:"domain_validation_method,omitempty"`
}

// ReissueSSLOrderResponse represents the API response for reissuing an SSL order.
type ReissueSSLOrderResponse struct {
	Code int      `json:"code"`
	Data SSLOrder `json:"data"`
}

// CancelSSLOrderResponse represents the API response for canceling an SSL order.
type CancelSSLOrderResponse struct {
	Code int    `json:"code"`
	Desc string `json:"desc"`
}

// ListSSLProductsResponse represents the API response for listing SSL products.
type ListSSLProductsResponse struct {
	Code int                          `json:"code"`
	Data ListSSLProductsResponseData  `json:"data"`
	Desc string                       `json:"desc"`
}

// ListSSLProductsResponseData contains the products list data.
type ListSSLProductsResponseData struct {
	Results []SSLProduct `json:"results"`
	Total   int          `json:"total"`
}

// GetSSLProductResponse represents the API response for getting an SSL product.
type GetSSLProductResponse struct {
	Code int        `json:"code"`
	Data SSLProduct `json:"data"`
}
