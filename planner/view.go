// Copyright 2023 Democratized Data Foundation
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

package planner

import (
	"github.com/sourcenetwork/defradb/client"
	"github.com/sourcenetwork/defradb/core"
	"github.com/sourcenetwork/defradb/planner/mapper"
)

// viewNode processes queries to a Defra View constructed from a base query ahead of time.
type viewNode struct {
	docMapper

	p      *Planner
	desc   client.CollectionDescription
	source planNode

	// This is cached as a boolean to save rediscovering this in the main Next/Value iteration loop
	hasTransform bool
}

func (p *Planner) View(query *mapper.Select, col client.Collection) (planNode, error) {
	// For now, we assume a single source.  This will need to change if/when we support multiple sources
	querySource := (col.Description().Sources[0].(*client.QuerySource))
	hasTransform := querySource.Transform.HasValue()

	m, err := mapper.ToSelect(p.ctx, p.db, &querySource.Query)
	if err != nil {
		return nil, err
	}

	source, err := p.Select(m)
	if err != nil {
		return nil, err
	}

	if hasTransform {
		source = p.Lens(source, query.DocumentMapping, col)
	}

	viewNode := &viewNode{
		p:            p,
		desc:         col.Description(),
		source:       source,
		docMapper:    docMapper{query.DocumentMapping},
		hasTransform: hasTransform,
	}

	return viewNode, nil
}

func (n *viewNode) Init() error {
	return n.source.Init()
}

func (n *viewNode) Start() error {
	return n.source.Start()
}

func (n *viewNode) Spans(spans core.Spans) {
	n.source.Spans(spans)
}

func (n *viewNode) Next() (bool, error) {
	return n.source.Next()
}

func (n *viewNode) Value() core.Doc {
	sourceValue := n.source.Value()
	if n.hasTransform {
		// If this view has a transform the source document will already have been
		// converted to the new document mapping.
		return sourceValue
	}

	sourceMap := n.source.DocumentMap().ToMap(sourceValue)

	// We must convert the document from the source mapping (which was constructed using the
	// view's base query) to a document using the output mapping (which was constructed using
	// the current query and the output schemas).  We do this by source output name, which
	// will take into account any aliases defined in the base query.
	doc := n.docMapper.documentMapping.NewDoc()
	for fieldName, fieldValue := range sourceMap {
		// If the field does not exist, ignore it an continue.  It likely means that
		// the field was declared in the query but not the SDL, and if it is not in the
		// SDL it cannot be requested/rendered by the user and would be dropped later anyway.
		_ = n.docMapper.documentMapping.TrySetFirstOfName(&doc, fieldName, fieldValue)
	}

	return doc
}

func (n *viewNode) Source() planNode {
	return n.source
}

func (n *viewNode) Kind() string {
	return "viewNode"
}

func (n *viewNode) Close() error {
	if n.source != nil {
		err := n.source.Close()
		if err != nil {
			return err
		}
	}

	return nil
}
