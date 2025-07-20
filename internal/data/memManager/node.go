package memManager

import (
	"context"
	"niurou/internal/data/graphDB"
)

func (m *managerImpl) AddPersonNode(ctx context.Context, personNode *graphDB.Person, labels []string) error {
	graphService := m.graphService
	graphService.AddPersonNode(ctx, personNode, labels)
	return nil
}
