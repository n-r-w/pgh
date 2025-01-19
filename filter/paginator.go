package filter

type paginatorType int

const (
	emptyPaginatorType = paginatorType(iota)
	frontPaginatorType
	servicePaginatorType
)

// Paginator Data pagination
type Paginator struct {
	limit  uint32
	offset uint64

	// Value of the last received id
	lastID        any
	paginatorType paginatorType
}

// NewFrontPaginator returns a frontend type paginator
func NewFrontPaginator(limit uint32, page uint64) *Paginator {
	return &Paginator{
		limit:         limit,
		offset:        (page - 1) * uint64(limit),
		paginatorType: frontPaginatorType,
	}
}

// NewServicePaginator returns a service type paginator
func NewServicePaginator(limit uint32, lastID uint64) *Paginator {
	return &Paginator{
		limit:         limit,
		lastID:        lastID,
		paginatorType: servicePaginatorType,
	}
}

// IsFront returns true if paginator type is frontend
func (p *Paginator) IsFront() bool {
	return p.paginatorType == frontPaginatorType
}

// IsService returns true if paginator type is service
func (p *Paginator) IsService() bool {
	return p.paginatorType == servicePaginatorType
}

func (p *Paginator) isEmpty() bool {
	return p.paginatorType == emptyPaginatorType
}
