// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package spicedb

import (
	"context"
	"fmt"
	"regexp"

	"github.com/authzed/spicedb/pkg/tuple"

	"github.com/authzed/authzed-go/v1"

	pb "github.com/authzed/authzed-go/proto/authzed/api/v1"

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
	fmt.Printf("new2 add policy as %#v\n", pr)
	r := fmt.Sprintf("%s:%s#%s@%s:%s", pr.ObjectType, pr.Object, pr.Relation, pr.SubjectType, pr.Subject)
	fmt.Println("request is ", r)
	req := &pb.WriteRelationshipsRequest{Updates: []*pb.RelationshipUpdate{{
		Operation:    pb.RelationshipUpdate_OPERATION_CREATE,
		Relationship: tuple.ParseRel(r),
	},
	}}
	_, err := pa.client.WriteRelationships(ctx, req)
	if err != nil {
		fmt.Println("err")
		return errors.Wrap(err, auth.ErrAuthorization)
	}
	fmt.Println("done")
	return nil
	// if isSubjectSet(pr.Subject) {
	// 	req := &pb.WriteRelationshipsRequest{Updates: []*pb.RelationshipUpdate{
	// 		{
	// 			Operation: pb.RelationshipUpdate_OPERATION_CREATE,
	//
	// 			Relationship: &pb.Relationship{
	// 				Resource: &pb.ObjectReference{ObjectType: pr.ObjectType, ObjectId: pr.Object},
	// 				Relation: pr.Relation,
	// 				Subject:  tuple.ParseRel(pr.Subject),
	// 			},
	// 		},
	// 	}}
	// 	_, err := pa.client.WriteRelationships(ctx, req)
	// 	if err != nil {
	// 		return errors.Wrap(err, auth.ErrAuthorization)
	// 	}
	// 	return nil
	// }
	//
	// req := &pb.WriteRelationshipsRequest{Updates: []*pb.RelationshipUpdate{
	// 	{
	// 		Operation: pb.RelationshipUpdate_OPERATION_CREATE,
	//
	// 		Relationship: &pb.Relationship{
	// 			Resource: &pb.ObjectReference{ObjectType: pr.ObjectType, ObjectId: pr.Object},
	// 			Relation: pr.Relation,
	// 			Subject:  getSubject(pr.SubjectType, pr.Subject),
	// 		},
	// 	},
	// }}
	// _, err := pa.client.WriteRelationships(ctx, req)
	// if err != nil {
	// 	return errors.Wrap(err, auth.ErrAuthorization)
	// }
	// return nil
}

func (pa policyAgent) DeletePolicy(ctx context.Context, pr auth.PolicyReq) error {
	// DeletePolicy is not implemented yet for SpiceDB.
	return nil
}

func getSubject(subjectType, subjectID string) *pb.SubjectReference {
	fmt.Println("incoming subject ", subjectID)
	if isSubjectSet(subjectID) {
		return &pb.SubjectReference{OptionalRelation: fmt.Sprintf("%s:%s", subjectType, subjectID)}
	}
	fmt.Println("not subject set")
	return &pb.SubjectReference{Object: &pb.ObjectReference{ObjectType: subjectType, ObjectId: subjectID}}
}

// isSubjectSet returns true when given subject is subject set.
// Otherwise, it returns false.
func isSubjectSet(subject string) bool {
	r, err := regexp.Compile(subjectSetRegex)
	if err != nil {
		return false
	}
	return r.MatchString(subject)
}
