"use client";

import React, { useCallback, useState } from 'react';
import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  SortingState,
  useReactTable,
} from '@tanstack/react-table';
import { Table, Form, Button, Modal, Spinner } from 'react-bootstrap';
import styles from './UserTable.module.css';
import ConfirmationModal from '../ConfirmationModal';
import { useApproveRequest, useDenyRequest, useKickMember, useUpdateUserRole } from '@/hook/request';
import { FaSearch, FaSort, FaSortDown, FaSortUp } from 'react-icons/fa';
import GrappleIcon from '../GrappleIcon';
import { useGetMembershipPlans } from '@/hook/membershipPlan';
import { useAssignPlan } from '@/hook/memberBilling';
import { useGetMembersRoster } from '@/hook/members';
import { RichMember } from '@/api-requests/members';
import MemberDrawer from '@/components/MemberDrawer';
import BeltBadge from '@/components/BeltBadge';

// User is kept for backward compat with MemberDrawer and other consumers.
// RichMember is a superset, so we alias it here.
export type User = RichMember;

export interface Filters {
  label: string;
  value: string;
  field: string;
}

interface Props {
  initialData: User[];
  setShow: (show: boolean) => void;
  filters: Filters[];
  gym: any;
  role: string;
}

const UsersTable = ({
  setShow,
  filters,
}: Props) => {
  const [selectedFilters, setSelectedFilters] = useState<Filters[]>([]);
  const [activeFilterParams, setActiveFilterParams] = useState<Record<string, string>>({});
  const [updateData, setUpdateData] = useState({
    role: '',
    username: '',
    cognito_id: '',
  });
  const [showConfirmationModal, setShowConfirmationModal] = useState(false);
  const [showConfirmKickModal, setShowConfirmKickModal] = useState(false);
  const [actionsData, setActionsData] = useState({
    newAction: '',
    data: '',
  });
  const [searchTerm, setSearchTerm] = useState('');
  const [sorting, setSorting] = useState<SortingState>([]);
  const [pageSize, setPageSize] = useState(10);

  const [assignTarget, setAssignTarget] = useState<{ memberId: string; memberName: string } | null>(null);
  const [selectedPlanId, setSelectedPlanId] = useState('');
  const [drawerMember, setDrawerMember] = useState<User | null>(null);

  // ── Data ─────────────────────────────────────────────────────────────────
  const membersQuery = useGetMembersRoster(activeFilterParams);
  const members = membersQuery.data?.data ?? [];

  const updateRole = useUpdateUserRole();
  const plans = useGetMembershipPlans();
  const assignPlan = useAssignPlan();
  const approveMutation = useApproveRequest();
  const denyMutation = useDenyRequest();
  const kickMutation = useKickMember();

  const handleApprove = useCallback((id: string) => {
    approveMutation.mutate(id);
  }, [approveMutation]);

  const handleDeny = useCallback((id: string) => {
    denyMutation.mutate(id);
  }, [denyMutation]);

  // ── Filter / search ───────────────────────────────────────────────────────
  const handleFilterClick = (filter: Filters) => {
    setSelectedFilters(prev => {
      const alreadyActive = prev.some(f => f.value === filter.value);
      if (alreadyActive) {
        setActiveFilterParams({});
        return [];
      } else {
        setActiveFilterParams({ [filter.field]: filter.value });
        return [filter];
      }
    });
  };

  const onSearch = () => {
    setActiveFilterParams(prev => ({ ...prev, search: searchTerm }));
  };

  const clear = () => {
    setSelectedFilters([]);
    setSearchTerm('');
    setActiveFilterParams({});
  };

  // ── Table ─────────────────────────────────────────────────────────────────
  const columnHelper = createColumnHelper<User>();

  const columns = [
    {
      id: 'select',
      header: ({ table }: any) => (
        <Form.Check
          type="checkbox"
          checked={table.getIsAllRowsSelected()}
          onChange={table.getToggleAllRowsSelectedHandler()}
        />
      ),
      cell: ({ row }: any) => (
        <Form.Check
          type="checkbox"
          checked={row.getIsSelected()}
          onChange={row.getToggleSelectedHandler()}
        />
      ),
    },
    columnHelper.accessor(row => `${row.first_name} ${row.last_name}`, {
      id: 'profile',
      header: 'Profile',
      cell: info => (
        <div
          className={styles.userInfo}
          style={{ cursor: 'pointer' }}
          onClick={() => setDrawerMember(info.row.original)}
        >
          <img
            src={info.row.original.profile?.avatar_url}
            alt={info.getValue()}
            className={styles.profileImage}
          />
          <span className={styles.userName} style={{ textDecoration: 'underline', textDecorationColor: '#ccc' }}>
            {info.getValue()}
          </span>
        </div>
      ),
    }),
    columnHelper.group({
      id: 'contact',
      header: 'Contact',
      cell: info => (
        <div className={styles.contactInfo}>
          <span className={styles.email}>{info.row.original.requestor_email}</span>
          <span className={styles.phone}>{info.row.original?.profile?.phone_number}</span>
        </div>
      ),
    }),
    columnHelper.accessor('status', {
      header: 'Status',
      cell: info => (
        <div className={styles.contactInfo}>
          <span className={styles.email}>{info.row.original.status}</span>
        </div>
      ),
    }),
    columnHelper.accessor('membership_type', {
      header: 'Membership type',
      cell: info => (
        <span className={styles.membershipBadge}>
          {info.row.original.membership_type?.toLowerCase()}
        </span>
      ),
    }),
    columnHelper.accessor('requestor_id', {
      id: 'belt',
      header: 'Belt',
      cell: info => {
        const belt = info.row.original.current_belt;
        if (!belt) return <span style={{ fontSize: 12, color: '#aaa' }}>—</span>;
        return (
          <BeltBadge
            system={belt.system}
            belt={belt.belt}
            stripes={belt.stripes}
          />
        );
      },
    }),
    columnHelper.accessor('role', {
      header: 'Role',
      cell: info => {
        const isAccepted = info.row.original.status === 'Accepted';
        const currentRole = info.row.original.role?.toLowerCase() ?? '';

        return (
          <>
            {isAccepted ? (
              <Form.Select
                value={currentRole}
                onChange={(e) => {
                  setUpdateData({
                    username: info.row.original.requestor_email,
                    role: e.target.value,
                    cognito_id: info.row.original.requestor_id,
                  });
                  setShowConfirmationModal(true);
                }}
                className={styles.roleSelect}
              >
                <option value="coach">Coach</option>
                <option value="owner">Owner</option>
                <option value="student">Student</option>
              </Form.Select>
            ) : 'Not Approved Yet'}
          </>
        );
      },
    }),
    columnHelper.group({
      id: 'actions',
      header: () => 'Actions',
      cell: info => {
        const kick = info.row.original.status === 'Accepted';
        const memberName = `${info.row.original.first_name} ${info.row.original.last_name}`;
        return (
          <>
            {!kick && (
              <Button variant='dark' size='sm' onClick={() => {
                setActionsData({ newAction: 'Approve', data: info.row.original.id });
                setShowConfirmKickModal(true);
              }}>
                Approve
              </Button>
            )}
            {kick && (
              <Button
                variant='outline-dark'
                size='sm'
                style={{ marginRight: 6 }}
                onClick={() => {
                  setSelectedPlanId('');
                  setAssignTarget({ memberId: info.row.original.requestor_id, memberName });
                }}
              >
                Assign Plan
              </Button>
            )}
            <Button
              style={{ marginLeft: kick ? 0 : 10, backgroundColor: 'white', color: 'black' }}
              variant='dark'
              size='sm'
              onClick={() => {
                setActionsData({ newAction: kick ? 'Kick' : 'Deny', data: info.row.original.id });
                setShowConfirmKickModal(true);
              }}
            >
              {kick ? 'Kick' : 'Deny'}
            </Button>
          </>
        );
      },
    }),
  ];

  const table = useReactTable({
    data: members,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    state: {
      sorting,
      pagination: { pageSize, pageIndex: 0 },
    },
    onSortingChange: setSorting,
  });

  return (
    <div>
      <div className={styles.tableControls}>
        <div className={styles.topButtons}>
          <div className={styles.button} onClick={() => setShow(true)}>
            <GrappleIcon src='/upload-email.svg' variant='dark' /> Upload Emails
          </div>
        </div>
        <div className={styles.searchContainer}>
          <div className={styles.searchInputWrapper}>
            <FaSearch className={styles.searchIcon} />
            <Form.Control
              type="search"
              placeholder="Search..."
              value={searchTerm}
              onChange={e => setSearchTerm(e.target.value)}
              onKeyDown={e => e.key === 'Enter' && onSearch()}
            />
          </div>
          <Button variant="dark" className={styles.searchButton} onClick={onSearch}>
            Search
          </Button>
          <Button variant="dark" className={styles.searchButton} onClick={clear}>
            Clear
          </Button>
        </div>
        <div className={styles.filterContainer}>
          {filters.map((filter: Filters) => (
            <button
              key={filter.value}
              onClick={() => handleFilterClick(filter)}
              className={`${styles.filterItem} ${
                selectedFilters.some(f => f.value === filter.value) ? styles.active : ''
              }`}
            >
              {filter.label}
            </button>
          ))}
        </div>
      </div>

      {membersQuery.isPending ? (
        <div style={{ display: 'flex', justifyContent: 'center', padding: '60px 0' }}>
          <Spinner animation="border" variant="dark" />
        </div>
      ) : (
        <div className={styles.tableWrapper}>
          <Table className={styles.table}>
            <thead>
              {table.getHeaderGroups().map(headerGroup => (
                <tr key={headerGroup.id}>
                  {headerGroup.headers.map(header => (
                    <th
                      key={header.id}
                      onClick={header.column.getToggleSortingHandler()}
                      className={styles.sortableHeader}
                    >
                      <div className={styles.headerContent}>
                        {flexRender(header.column.columnDef.header, header.getContext())}
                        {header.column.getCanSort() && (
                          <span className={styles.sortIcon}>
                            {header.column.getIsSorted() === "asc" ? (
                              <FaSortUp />
                            ) : header.column.getIsSorted() === "desc" ? (
                              <FaSortDown />
                            ) : (
                              <FaSort />
                            )}
                          </span>
                        )}
                      </div>
                    </th>
                  ))}
                </tr>
              ))}
            </thead>
            <tbody>
              {table.getRowModel().rows.map(row => (
                <tr key={row.id}>
                  {row.getVisibleCells().map(cell => (
                    <td key={cell.id} style={{ verticalAlign: 'middle' }}>
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </Table>
          <div className={styles.paginationControls}>
            <div className={styles.pageSizeSelector}>
              Show
              <Form.Select
                value={pageSize}
                onChange={e => setPageSize(Number(e.target.value))}
                className={styles.pageSizeSelect}
              >
                {[10, 25, 50].map(size => (
                  <option key={size} value={size}>{size}</option>
                ))}
              </Form.Select>
              entries
            </div>
            <div className={styles.pagination}>
              <Button onClick={() => table.setPageIndex(0)} disabled={!table.getCanPreviousPage()} variant="outline-dark">{'<<'}</Button>
              <Button onClick={() => table.previousPage()} disabled={!table.getCanPreviousPage()} variant="outline-dark">{'<'}</Button>
              <span className={styles.pageInfo}>
                Page <strong>{table.getState().pagination.pageIndex + 1} of {table.getPageCount()}</strong>
              </span>
              <Button onClick={() => table.nextPage()} disabled={!table.getCanNextPage()} variant="outline-dark">{'>'}</Button>
              <Button onClick={() => table.setPageIndex(table.getPageCount() - 1)} disabled={!table.getCanNextPage()} variant="outline-dark">{'>>'}</Button>
            </div>
          </div>
        </div>
      )}

      <ConfirmationModal
        show={showConfirmationModal}
        setShow={setShowConfirmationModal}
        onConfirm={() => {
          updateRole.mutate(updateData);
          setShowConfirmationModal(false);
        }}
      >
        Are you sure you want to update the role of {updateData.username} to {updateData.role}?
      </ConfirmationModal>

      <ConfirmationModal
        show={showConfirmKickModal}
        setShow={setShowConfirmKickModal}
        onConfirm={() => {
          if (actionsData.newAction === 'Approve') {
            handleApprove(actionsData.data);
          } else if (actionsData.newAction === 'Kick') {
            kickMutation.mutate(actionsData.data);
          } else {
            handleDeny(actionsData.data);
          }
          setShowConfirmKickModal(false);
        }}
      >
        Are you sure you want to {actionsData.newAction.toLowerCase()} this user?
      </ConfirmationModal>

      <MemberDrawer
        member={drawerMember}
        onClose={() => setDrawerMember(null)}
      />

      <Modal show={!!assignTarget} onHide={() => setAssignTarget(null)} centered>
        <Modal.Header closeButton style={{ background: '#000', color: '#fff' }}>
          <Modal.Title style={{ fontSize: 16 }}>
            Assign Plan — {assignTarget?.memberName}
          </Modal.Title>
        </Modal.Header>
        <Modal.Body style={{ padding: 24 }}>
          <Form.Group>
            <Form.Label>Membership Plan</Form.Label>
            <Form.Select
              value={selectedPlanId}
              onChange={e => setSelectedPlanId(e.target.value)}
            >
              <option value="">Select a plan...</option>
              {(plans.data ?? []).filter(p => p.is_active).map(p => (
                <option key={p.id} value={p.id}>
                  {p.name} — ${((p.price ?? 0) / 100).toFixed(2)}{p.billing_type === 'recurring' ? `/${p.interval}` : ' one-time'}
                </option>
              ))}
            </Form.Select>
          </Form.Group>
        </Modal.Body>
        <Modal.Footer>
          <Button variant="outline-secondary" onClick={() => setAssignTarget(null)}>Cancel</Button>
          <Button
            variant="dark"
            disabled={!selectedPlanId || assignPlan.isPending}
            onClick={() => {
              if (!assignTarget || !selectedPlanId) return;
              assignPlan.mutate(
                { member_id: assignTarget.memberId, plan_id: selectedPlanId, member_name: assignTarget.memberName },
                { onSuccess: () => setAssignTarget(null) }
              );
            }}
          >
            {assignPlan.isPending ? <Spinner size="sm" /> : 'Assign'}
          </Button>
        </Modal.Footer>
      </Modal>
    </div>
  );
};

export default UsersTable;
