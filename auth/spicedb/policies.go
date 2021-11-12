// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package spicedb

import (
	"context"
	"fmt"
	"io"
	"strings"

	pb "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/authzed/authzed-go/v1"
	"github.com/authzed/spicedb/pkg/tuple"
	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/pkg/errors"
)

const (
	subjectSetRegex = "^.{1,}#.{1,}$" // expected subject set structure is <namespace>:<object>#<relation>
)

type policyAgent struct {
	client *authzed.Client
}

// NewPolicyAgent returns a gRPC communication functionalities
// to communicate with SpiceDB.
func NewPolicyAgent(client *authzed.Client) auth.PolicyAgent {
	return policyAgent{client}
}

func (pa policyAgent) CheckPolicy(ctx context.Context, pr auth.PolicyReq) error {
	subject := &pb.SubjectReference{Object: &pb.ObjectReference{
		ObjectType: pr.SubjectType,
		ObjectId:   pr.Subject,
	}}
	object := &pb.ObjectReference{ObjectType: pr.ObjectType, ObjectId: pr.Object}
	resp, err := pa.client.CheckPermission(ctx, &pb.CheckPermissionRequest{
		Resource:   object,
		Permission: pr.Relation,
		Subject:    subject,
	})
	if err != nil {
		return errors.Wrap(err, auth.ErrAuthorization)
	}
	if resp.GetPermissionship() != pb.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION {
		return auth.ErrAuthorization
	}
	return nil
}

func (pa policyAgent) AddPolicy(ctx context.Context, pr auth.PolicyReq) error {
	req := &pb.WriteRelationshipsRequest{Updates: []*pb.RelationshipUpdate{{
		Operation:    pb.RelationshipUpdate_OPERATION_CREATE,
		Relationship: tuple.ParseRel(fmt.Sprintf("%s:%s#%s@%s:%s", pr.ObjectType, pr.Object, pr.Relation, pr.SubjectType, pr.Subject)),
	},
	}}
	_, err := pa.client.WriteRelationships(ctx, req)
	if err != nil {
		return errors.Wrap(err, auth.ErrAuthorization)
	}
	return nil
}

// DeletePolicy is not implemented! Instead, it just printing the expanding results of the policies.
func (pa policyAgent) DeletePolicy(ctx context.Context, pr auth.PolicyReq) error {
	objectNS := pr.ObjectType
	relation := pr.Relation
	subjectNS := pr.SubjectType
	subjectID, subjectRel := parseSubject(pr.Subject)

	request := &pb.LookupResourcesRequest{
		ResourceObjectType: objectNS,
		Permission:         relation,
		Subject: &pb.SubjectReference{
			Object: &pb.ObjectReference{
				ObjectType: subjectNS,
				ObjectId:   subjectID,
			},
			OptionalRelation: subjectRel,
		},
	}
	respStream, err := pa.client.LookupResources(context.Background(), request)
	if err != nil {
		return fmt.Errorf("failed to create lookupresource stream")
	}

	counter := 0
	for {
		r, err := respStream.Recv()
		switch {
		case err == io.EOF:
			fmt.Println("DONE/EOF")
			return nil
		case err != nil:
			fmt.Println("DONE")
			return err
		default:
			i := r.ResourceObjectId
			if i == "" {
				fmt.Println("FINISHED")
				return nil
			}
			fmt.Printf("%d\t%s\n", counter, i)
		}
		counter++
	}
}

func parseSubject(subject string) (id, relation string) {
	sarr := strings.Split(subject, "#")
	if len(sarr) != 2 {
		return subject, ""
	}
	return sarr[0], sarr[1]
}
