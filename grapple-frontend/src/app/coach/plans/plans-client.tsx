'use client';

import { useState } from 'react';
import { Button, Card, Col, Form, Modal, Row, Spinner, Badge } from 'react-bootstrap';
import { useGetMembershipPlans, useCreateMembershipPlan, useUpdateMembershipPlan, useDeleteMembershipPlan } from '@/hook/membershipPlan';
import { MembershipPlan } from '@/api-requests/membershipPlan';
import { useGetGym } from '@/hook/gym';
import ConfirmationModal from '@/components/ConfirmationModal';
import styles from './Plans.module.css';

const EMPTY_FORM: Omit<MembershipPlan, 'id' | 'is_active' | 'created_at' | 'updated_at'> = {
  gym_id: '',
  name: '',
  description: '',
  billing_type: 'recurring',
  interval: 'monthly',
  price: 0,
  currency: 'usd',
  class_limit: null,
};

function formatPrice(cents: number, currency = 'usd') {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: currency.toUpperCase(),
  }).format(cents / 100);
}

function PlanFormModal({
  show,
  onHide,
  onSubmit,
  initial,
  isLoading,
}: {
  show: boolean;
  onHide: () => void;
  onSubmit: (plan: any) => void;
  initial?: Partial<MembershipPlan>;
  isLoading: boolean;
}) {
  const [form, setForm] = useState({
    name: initial?.name ?? '',
    description: initial?.description ?? '',
    billing_type: initial?.billing_type ?? 'recurring',
    interval: initial?.interval ?? 'monthly',
    price: initial?.price != null ? (initial.price / 100).toFixed(2) : '',
    class_limit: initial?.class_limit != null ? String(initial.class_limit) : '',
  });

  const set = (field: string, value: any) => setForm(prev => ({ ...prev, [field]: value }));

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit({
      ...form,
      price: Math.round(parseFloat(form.price) * 100),
      class_limit: form.class_limit !== '' ? parseInt(form.class_limit) : null,
      currency: 'usd',
    });
  };

  return (
    <Modal show={show} onHide={onHide} centered>
      <Modal.Header closeButton className={styles.modalHeader}>
        <Modal.Title>{initial?.id ? 'Edit Plan' : 'New Plan'}</Modal.Title>
      </Modal.Header>
      <Modal.Body className={styles.modalBody}>
        <Form onSubmit={handleSubmit}>
          <Form.Group className="mb-3">
            <Form.Label>Plan Name</Form.Label>
            <Form.Control
              required
              placeholder="e.g. Monthly Unlimited"
              value={form.name}
              onChange={e => set('name', e.target.value)}
            />
          </Form.Group>

          <Form.Group className="mb-3">
            <Form.Label>Description</Form.Label>
            <Form.Control
              as="textarea"
              rows={2}
              placeholder="Optional description"
              value={form.description}
              onChange={e => set('description', e.target.value)}
            />
          </Form.Group>

          <Row className="mb-3">
            <Col>
              <Form.Label>Billing Type</Form.Label>
              <Form.Select value={form.billing_type} onChange={e => set('billing_type', e.target.value)}>
                <option value="recurring">Recurring</option>
                <option value="one_time">One-Time</option>
              </Form.Select>
            </Col>
            {form.billing_type === 'recurring' && (
              <Col>
                <Form.Label>Interval</Form.Label>
                <Form.Select value={form.interval} onChange={e => set('interval', e.target.value)}>
                  <option value="monthly">Monthly</option>
                  <option value="yearly">Yearly</option>
                  <option value="weekly">Weekly</option>
                </Form.Select>
              </Col>
            )}
          </Row>

          <Row className="mb-3">
            <Col>
              <Form.Label>Price (USD)</Form.Label>
              <Form.Control
                required
                type="number"
                min="0"
                step="0.01"
                placeholder="0.00"
                value={form.price}
                onChange={e => set('price', e.target.value)}
              />
            </Col>
            <Col>
              <Form.Label>Class Limit</Form.Label>
              <Form.Control
                type="number"
                min="1"
                placeholder="Unlimited"
                value={form.class_limit}
                onChange={e => set('class_limit', e.target.value)}
              />
              <Form.Text className="text-muted">Leave blank for unlimited</Form.Text>
            </Col>
          </Row>

          <div className="d-flex justify-content-end gap-2">
            <Button variant="outline-secondary" onClick={onHide} type="button">
              Cancel
            </Button>
            <Button variant="dark" type="submit" disabled={isLoading}>
              {isLoading ? <Spinner size="sm" /> : (initial?.id ? 'Save Changes' : 'Create Plan')}
            </Button>
          </div>
        </Form>
      </Modal.Body>
    </Modal>
  );
}

export default function PlansClient() {
  const gym = useGetGym();
  const gymId = gym?.data?.id;

  const plans = useGetMembershipPlans();
  const createPlan = useCreateMembershipPlan();
  const updatePlan = useUpdateMembershipPlan();
  const deletePlan = useDeleteMembershipPlan();

  const [showModal, setShowModal] = useState(false);
  const [editingPlan, setEditingPlan] = useState<MembershipPlan | undefined>(undefined);
  const [deletingPlanId, setDeletingPlanId] = useState<string | null>(null);

  const handleCreate = (form: any) => {
    createPlan.mutate({ ...form, gym_id: gymId }, {
      onSuccess: () => setShowModal(false),
    });
  };

  const handleUpdate = (form: any) => {
    if (!editingPlan?.id) return;
    updatePlan.mutate({ planId: editingPlan.id, updates: form }, {
      onSuccess: () => setEditingPlan(undefined),
    });
  };

  const handleDelete = () => {
    if (!deletingPlanId) return;
    deletePlan.mutate(deletingPlanId, {
      onSuccess: () => setDeletingPlanId(null),
    });
  };

  const activePlans = (plans.data ?? []).filter(p => p.is_active);
  const inactivePlans = (plans.data ?? []).filter(p => !p.is_active);

  if (plans.isPending) {
    return (
      <div className={styles.loadingWrapper}>
        <Spinner animation="border" variant="dark" />
      </div>
    );
  }

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <div>
          <h4 className={styles.title}>Membership Plans</h4>
          <p className={styles.subtitle}>Create and manage the plans you offer to members.</p>
        </div>
        <Button variant="dark" onClick={() => setShowModal(true)}>
          + New Plan
        </Button>
      </div>

      {activePlans.length === 0 && (
        <div className={styles.emptyState}>
          <p>No plans yet. Create your first membership plan to get started.</p>
        </div>
      )}

      <Row className="g-3">
        {activePlans.map(plan => (
          <Col key={plan.id} xs={12} md={6} lg={4}>
            <Card className={styles.planCard}>
              <Card.Body>
                <div className={styles.planCardHeader}>
                  <div>
                    <Card.Title className={styles.planName}>{plan.name}</Card.Title>
                    <Badge bg="secondary" className={styles.badge}>
                      {plan.billing_type === 'recurring' ? plan.interval : 'one-time'}
                    </Badge>
                  </div>
                  <div className={styles.price}>
                    {formatPrice(plan.price, plan.currency)}
                  </div>
                </div>

                {plan.description && (
                  <p className={styles.description}>{plan.description}</p>
                )}

                <div className={styles.planMeta}>
                  <span>{plan.class_limit == null ? 'Unlimited classes' : `${plan.class_limit} classes`}</span>
                </div>

                <div className={styles.planActions}>
                  <Button
                    variant="outline-dark"
                    size="sm"
                    onClick={() => setEditingPlan(plan)}
                  >
                    Edit
                  </Button>
                  <Button
                    variant="outline-danger"
                    size="sm"
                    onClick={() => setDeletingPlanId(plan.id!)}
                  >
                    Remove
                  </Button>
                </div>
              </Card.Body>
            </Card>
          </Col>
        ))}
      </Row>

      {inactivePlans.length > 0 && (
        <div className={styles.inactiveSection}>
          <p className={styles.inactiveLabel}>Inactive Plans ({inactivePlans.length})</p>
        </div>
      )}

      {showModal && (
        <PlanFormModal
          show={showModal}
          onHide={() => setShowModal(false)}
          onSubmit={handleCreate}
          isLoading={createPlan.isPending}
        />
      )}

      {editingPlan && (
        <PlanFormModal
          show={!!editingPlan}
          onHide={() => setEditingPlan(undefined)}
          onSubmit={handleUpdate}
          initial={editingPlan}
          isLoading={updatePlan.isPending}
        />
      )}

      <ConfirmationModal
        show={!!deletingPlanId}
        setShow={(v: boolean) => { if (!v) setDeletingPlanId(null); }}
        onConfirm={handleDelete}
      >
        Are you sure you want to remove this plan? Members already on this plan will not be affected.
      </ConfirmationModal>
    </div>
  );
}
