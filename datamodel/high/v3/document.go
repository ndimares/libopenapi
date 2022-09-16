// Copyright 2022 Princess B33f Heavy Industries / Dave Shanley
// SPDX-License-Identifier: MIT

// Package v3 represents all OpenAPI 3+ high-level models. High-level models are easy to navigate
// and simple to extract what ever is required from an OpenAPI 3+ specification.
//
// High-level models are backed by low-level ones. There is a 'GoLow()' method available on every high level
// object. 'Going Low' allows engineers to transition from a high-level or 'porcelain' API, to a low-level 'plumbing'
// API, which provides fine grain detail to the underlying AST powering the data, lines, columns, raw nodes etc.
package v3

import (
	"github.com/pb33f/libopenapi/datamodel/high"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	low "github.com/pb33f/libopenapi/datamodel/low/v3"
	"github.com/pb33f/libopenapi/index"
)

// Document represents a high-level OpenAPI 3 document (both 3.0 & 3.1). A Document is the root of the specification.
type Document struct {

	// Version is the version of OpenAPI being used, extracted from the 'openapi: x.x.x' definition.
	// This is not a standard property of the OpenAPI model, it's a convenience mechanism only.
	Version string

	// Info presents a specification Info definitions
	// - https://spec.openapis.org/oas/v3.1.0#info-object
	Info *base.Info

	// Servers is a slice of Server instances
	// - https://spec.openapis.org/oas/v3.1.0#server-object
	Servers []*Server

	// Paths contains all the PathItem definitions for the specification.
	// - https://spec.openapis.org/oas/v3.1.0#paths-object
	Paths *Paths

	// Components contains everything defined as a component (referenced by everything else)
	// - https://spec.openapis.org/oas/v3.1.0#components-object
	Components *Components

	// Security contains global security requirements/roles for the specification
	// - https://spec.openapis.org/oas/v3.1.0#security-requirement-object
	Security *SecurityRequirement

	// Tags is a slice of base.Tag instances defined by the specification
	// - https://spec.openapis.org/oas/v3.1.0#tag-object
	Tags []*base.Tag

	// ExternalDocs is an instance of base.ExternalDoc for.. well, obvious really, innit.
	// - https://spec.openapis.org/oas/v3.1.0#external-documentation-object
	ExternalDocs *base.ExternalDoc

	// Extensions contains all custom extensions defined for the top-level document.
	Extensions map[string]any

	// JsonSchemaDialect is a 3.1+ property that sets the dialect to use for validating *base.Schema definitions
	// - https://spec.openapis.org/oas/v3.1.0#schema-object
	JsonSchemaDialect string

	// Webhooks is a 3.1+ property that is similar to callbacks, except, this defines incoming webhooks.
	Webhooks map[string]*PathItem

	// Index is a reference to the *index.SpecIndex that was created for the document and used
	// as a guide when building out the Document. Ideal if further processing is required on the model and
	// the original details are required to continue the work.
	//
	// This property is not a part of the OpenAPI schema, this is custom to libopenapi.
	Index *index.SpecIndex
	low   *low.Document
}

// NewDocument will create a new high-level Document from a low-level one.
func NewDocument(document *low.Document) *Document {
	d := new(Document)
	d.low = document
	d.Index = document.Index
	if !document.Info.IsEmpty() {
		d.Info = base.NewInfo(document.Info.Value)
	}
	if !document.Version.IsEmpty() {
		d.Version = document.Version.Value
	}
	var servers []*Server
	for _, ser := range document.Servers.Value {
		servers = append(servers, NewServer(ser.Value))
	}
	d.Servers = servers
	var tags []*base.Tag
	for _, tag := range document.Tags.Value {
		tags = append(tags, base.NewTag(tag.Value))
	}
	d.Tags = tags
	if !document.ExternalDocs.IsEmpty() {
		d.ExternalDocs = base.NewExternalDoc(document.ExternalDocs.Value)
	}
	if len(document.Extensions) > 0 {
		d.Extensions = high.ExtractExtensions(document.Extensions)
	}
	if !document.Components.IsEmpty() {
		d.Components = NewComponents(document.Components.Value)
	}
	if !document.Paths.IsEmpty() {
		d.Paths = NewPaths(document.Paths.Value)
	}
	if !document.JsonSchemaDialect.IsEmpty() {
		d.JsonSchemaDialect = document.JsonSchemaDialect.Value
	}
	if !document.Webhooks.IsEmpty() {
		hooks := make(map[string]*PathItem)
		for h := range document.Webhooks.Value {
			hooks[h.Value] = NewPathItem(document.Webhooks.Value[h].Value)
		}
		d.Webhooks = hooks
	}
	return d
}

func (d *Document) GoLow() *low.Document {
	return d.low
}