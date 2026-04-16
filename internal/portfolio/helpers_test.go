package portfolio

import (
	"github.com/project/vk-investment-middleend/internal/components"
)

func findDescendantByID(c components.Component, id string) *components.Component {
	if c.ID == id {
		return &c
	}
	for i := range c.Children {
		if found := findDescendantByID(c.Children[i], id); found != nil {
			return found
		}
	}
	return nil
}
