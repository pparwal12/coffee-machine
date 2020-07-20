package entities

type ErrResourceTemporarilyNotAvailable struct {
	ResourceID string
}

func (e ErrResourceTemporarilyNotAvailable) Error() string {
	return "resource temporarily unavailable, resource-id : " + e.ResourceID
}

type ErrInsufficientResource struct {
	ResourceID string
}

func (e ErrInsufficientResource) Error() string {
	return "resource is insufficient, resource-id : " + e.ResourceID
}

type ErrResourceNotAvailable struct {
	ResourceID string
}

func (e ErrResourceNotAvailable) Error() string {
	return "resource not available, resource-id : " + e.ResourceID
}
