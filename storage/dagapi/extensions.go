package dagapi

func (nd *Node) SearchLinkByName(name string) *Link {
	for _, link := range nd.Links {
		if link.Name == name {
			return link
		}
	}
	return nil
}
