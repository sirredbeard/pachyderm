package pfs

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pachyderm/pachyderm/v2/src/internal/require"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestCommit_NilBranch(t *testing.T) {
	var b1 = &Branch{Name: "dummy"}
	var c1 = &Commit{Branch: b1}
	c1.NilBranch()
	require.Nil(t, c1.Branch)

	var b2 = &Branch{Name: ""}
	var c2 = &Commit{Branch: b2}
	c2.NilBranch()
	require.Nil(t, c2.Branch)
}

func TestProject_ValidateName(t *testing.T) {
	var p = &Project{Name: "0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF"}
	err := p.ValidateName()
	require.YesError(t, err)
	require.True(t, strings.Contains(err.Error(), fmt.Sprintf("is %d characters", len(p.Name)-projectNameLimit)), fmt.Sprintf("missing %d", len(p.Name)-projectNameLimit))
}

func TestUnmarshalProjectPicker(t *testing.T) {
	var p ProjectPicker
	if err := p.UnmarshalText([]byte(DefaultProjectName)); err != nil {
		t.Fatal(err)
	}
	require.NoDiff(t, &p, &ProjectPicker{
		Picker: &ProjectPicker_Name{
			Name: DefaultProjectName,
		},
	}, []cmp.Option{protocmp.Transform()})
}

func TestUnmarshalRepoPicker(t *testing.T) {
	testData := []struct {
		name  string
		input string
		want  *RepoPicker
	}{
		{
			name:  "user repo",
			input: "default/images",
			want: &RepoPicker{
				Picker: &RepoPicker_Name{
					Name: &RepoPicker_RepoName{
						Project: &ProjectPicker{
							Picker: &ProjectPicker_Name{
								Name: "default",
							},
						},
						Name: "images",
						Type: "user",
					},
				},
			},
		},
		{
			name:  "user repo without project",
			input: "images",
			want: &RepoPicker{
				Picker: &RepoPicker_Name{
					Name: &RepoPicker_RepoName{
						Name:    "images",
						Type:    "user",
						Project: &ProjectPicker{},
					},
				},
			},
		},
		{
			name:  "spec repo",
			input: "default/images.spec",
			want: &RepoPicker{
				Picker: &RepoPicker_Name{
					Name: &RepoPicker_RepoName{
						Project: &ProjectPicker{
							Picker: &ProjectPicker_Name{
								Name: "default",
							},
						},
						Name: "images",
						Type: "spec",
					},
				},
			},
		},
		{
			name:  "spec repo without project",
			input: "images.spec",
			want: &RepoPicker{
				Picker: &RepoPicker_Name{
					Name: &RepoPicker_RepoName{
						Name:    "images",
						Type:    "spec",
						Project: &ProjectPicker{},
					},
				},
			},
		},
	}
	for _, test := range testData {
		t.Run(test.name, func(t *testing.T) {
			var p RepoPicker
			if err := p.UnmarshalText([]byte(test.input)); err != nil {
				t.Fatal(err)
			}
			require.NoDiff(t, &p, test.want, []cmp.Option{protocmp.Transform()})
		})
	}
}

func TestUnmarshalCommitPicker(t *testing.T) {
	testData := []struct {
		name    string
		input   string
		want    *CommitPicker
		wantErr bool
	}{
		{
			name:  "global id in user repo",
			input: "default/images@4444444444444444444444444444444A",
			want: &CommitPicker{
				Picker: &CommitPicker_Id{
					Id: &CommitPicker_CommitByGlobalId{
						Repo: &RepoPicker{
							Picker: &RepoPicker_Name{
								Name: &RepoPicker_RepoName{
									Project: &ProjectPicker{
										Picker: &ProjectPicker_Name{
											Name: "default",
										},
									},
									Name: "images",
									Type: "user",
								},
							},
						},
						Id: "4444444444444444444444444444444a",
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "branch name in user repo",
			input: "default/images@master",
			want: &CommitPicker{
				Picker: &CommitPicker_BranchHead{
					BranchHead: &BranchPicker{
						Picker: &BranchPicker_Name{
							Name: &BranchPicker_BranchName{
								Repo: &RepoPicker{
									Picker: &RepoPicker_Name{
										Name: &RepoPicker_RepoName{
											Project: &ProjectPicker{
												Picker: &ProjectPicker_Name{
													Name: "default",
												},
											},
											Name: "images",
											Type: "user",
										},
									},
								},
								Name: "master",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "branch name in user repo without project",
			input: "images@master",
			want: &CommitPicker{
				Picker: &CommitPicker_BranchHead{
					BranchHead: &BranchPicker{
						Picker: &BranchPicker_Name{
							Name: &BranchPicker_BranchName{
								Repo: &RepoPicker{
									Picker: &RepoPicker_Name{
										Name: &RepoPicker_RepoName{
											Name:    "images",
											Type:    "user",
											Project: &ProjectPicker{},
										},
									},
								},
								Name: "master",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "global id in user repo without project",
			input: "images@44444444444444444444444444444444",
			want: &CommitPicker{
				Picker: &CommitPicker_Id{
					Id: &CommitPicker_CommitByGlobalId{
						Repo: &RepoPicker{
							Picker: &RepoPicker_Name{
								Name: &RepoPicker_RepoName{
									Name:    "images",
									Type:    "user",
									Project: &ProjectPicker{},
								},
							},
						},
						Id: "44444444444444444444444444444444",
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "global id in spec repo",
			input: "default/images.spec@44444444444444444444444444444444",
			want: &CommitPicker{
				Picker: &CommitPicker_Id{
					Id: &CommitPicker_CommitByGlobalId{
						Repo: &RepoPicker{
							Picker: &RepoPicker_Name{
								Name: &RepoPicker_RepoName{
									Project: &ProjectPicker{
										Picker: &ProjectPicker_Name{
											Name: "default",
										},
									},
									Name: "images",
									Type: "spec",
								},
							},
						},
						Id: "44444444444444444444444444444444",
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "ancestor of a branch head. simple case 1",
			input: "images@master^",
			want: &CommitPicker{
				Picker: &CommitPicker_Ancestor{
					Ancestor: &CommitPicker_AncestorOf{
						Start: &CommitPicker{
							Picker: &CommitPicker_BranchHead{
								BranchHead: &BranchPicker{
									Picker: &BranchPicker_Name{
										Name: &BranchPicker_BranchName{
											Repo: &RepoPicker{
												Picker: &RepoPicker_Name{
													Name: &RepoPicker_RepoName{
														Name:    "images",
														Type:    "user",
														Project: &ProjectPicker{},
													},
												},
											},
											Name: "master",
										},
									},
								},
							},
						},
						Offset: 1,
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "ancestor of a branch head. simple case 2",
			input: "images@master^12",
			want: &CommitPicker{
				Picker: &CommitPicker_Ancestor{
					Ancestor: &CommitPicker_AncestorOf{
						Start: &CommitPicker{
							Picker: &CommitPicker_BranchHead{
								BranchHead: &BranchPicker{
									Picker: &BranchPicker_Name{
										Name: &BranchPicker_BranchName{
											Repo: &RepoPicker{
												Picker: &RepoPicker_Name{
													Name: &RepoPicker_RepoName{
														Name:    "images",
														Type:    "user",
														Project: &ProjectPicker{},
													},
												},
											},
											Name: "master",
										},
									},
								},
							},
						},
						Offset: 12,
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "ancestor of a commit. simple case 3",
			input: "images@44444444444444444444444444444444^",
			want: &CommitPicker{
				Picker: &CommitPicker_Ancestor{
					Ancestor: &CommitPicker_AncestorOf{
						Start: &CommitPicker{
							Picker: &CommitPicker_Id{
								Id: &CommitPicker_CommitByGlobalId{
									Repo: &RepoPicker{
										Picker: &RepoPicker_Name{
											Name: &RepoPicker_RepoName{
												Name:    "images",
												Type:    "user",
												Project: &ProjectPicker{},
											},
										},
									},
									Id: "44444444444444444444444444444444",
								},
							},
						},
						Offset: 1,
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "branch root",
			input: "images@master.13",
			want: &CommitPicker{
				Picker: &CommitPicker_BranchRoot_{
					BranchRoot: &CommitPicker_BranchRoot{
						Branch: &BranchPicker{
							Picker: &BranchPicker_Name{
								Name: &BranchPicker_BranchName{
									Repo: &RepoPicker{
										Picker: &RepoPicker_Name{
											Name: &RepoPicker_RepoName{
												Project: &ProjectPicker{},
												Name:    "images",
												Type:    "user",
											},
										},
									},
									Name: "master",
								},
							},
						},
						Offset: 12,
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "branch root and ancestor of. simple valid case",
			input: "images@master.2^1",
			want: &CommitPicker{
				Picker: &CommitPicker_BranchRoot_{
					BranchRoot: &CommitPicker_BranchRoot{
						Branch: &BranchPicker{
							Picker: &BranchPicker_Name{
								Name: &BranchPicker_BranchName{
									Repo: &RepoPicker{
										Picker: &RepoPicker_Name{
											Name: &RepoPicker_RepoName{
												Project: &ProjectPicker{},
												Name:    "images",
												Type:    "user",
											},
										},
									},
									Name: "master",
								},
							},
						},
						Offset: 0,
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "branch root and ancestor of. complex valid case",
			input: "images@master.70^12",
			want: &CommitPicker{
				Picker: &CommitPicker_BranchRoot_{
					BranchRoot: &CommitPicker_BranchRoot{
						Branch: &BranchPicker{
							Picker: &BranchPicker_Name{
								Name: &BranchPicker_BranchName{
									Repo: &RepoPicker{
										Picker: &RepoPicker_Name{
											Name: &RepoPicker_RepoName{
												Project: &ProjectPicker{},
												Name:    "images",
												Type:    "user",
											},
										},
									},
									Name: "master",
								},
							},
						},
						Offset: 57,
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "branch root and ancestor of. invalid case",
			input: "images@master.1^2",
			want: &CommitPicker{
				Picker: &CommitPicker_BranchRoot_{},
			},
			wantErr: true,
		},
	}
	for _, test := range testData {
		t.Run(test.name, func(t *testing.T) {
			var p CommitPicker
			err := p.UnmarshalText([]byte(test.input))
			if test.wantErr {
				require.YesError(t, err)
			} else {
				require.NoError(t, err)
				require.NoDiff(t, &p, test.want, []cmp.Option{protocmp.Transform()})
			}
		})
	}
}

func TestUnmarshalBranchPicker(t *testing.T) {
	testData := []struct {
		name  string
		input string
		want  *BranchPicker
	}{
		{
			name:  "branch in user repo",
			input: "default/images@master",
			want: &BranchPicker{
				Picker: &BranchPicker_Name{
					Name: &BranchPicker_BranchName{
						Repo: &RepoPicker{
							Picker: &RepoPicker_Name{
								Name: &RepoPicker_RepoName{
									Project: &ProjectPicker{
										Picker: &ProjectPicker_Name{
											Name: "default",
										},
									},
									Name: "images",
									Type: "user",
								},
							},
						},
						Name: "master",
					},
				},
			},
		},
		{
			name:  "branch in spec repo",
			input: "default/images.spec@master",
			want: &BranchPicker{
				Picker: &BranchPicker_Name{
					Name: &BranchPicker_BranchName{
						Repo: &RepoPicker{
							Picker: &RepoPicker_Name{
								Name: &RepoPicker_RepoName{
									Project: &ProjectPicker{
										Picker: &ProjectPicker_Name{
											Name: "default",
										},
									},
									Name: "images",
									Type: "spec",
								},
							},
						},
						Name: "master",
					},
				},
			},
		},
		{
			name:  "branch without project",
			input: "images@master",
			want: &BranchPicker{
				Picker: &BranchPicker_Name{
					Name: &BranchPicker_BranchName{
						Repo: &RepoPicker{
							Picker: &RepoPicker_Name{
								Name: &RepoPicker_RepoName{
									Project: &ProjectPicker{},
									Name:    "images",
									Type:    "user",
								},
							},
						},
						Name: "master",
					},
				},
			},
		},
		{
			name:  "branch in spec repo without project",
			input: "images.spec@master",
			want: &BranchPicker{
				Picker: &BranchPicker_Name{
					Name: &BranchPicker_BranchName{
						Repo: &RepoPicker{
							Picker: &RepoPicker_Name{
								Name: &RepoPicker_RepoName{
									Project: &ProjectPicker{},
									Name:    "images",
									Type:    "spec",
								},
							},
						},
						Name: "master",
					},
				},
			},
		},
	}
	for _, test := range testData {
		t.Run(test.name, func(t *testing.T) {
			var p BranchPicker
			if err := p.UnmarshalText([]byte(test.input)); err != nil {
				t.Fatal(err)
			}
			require.NoDiff(t, &p, test.want, []cmp.Option{protocmp.Transform()})
		})
	}
}
