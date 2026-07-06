<template>
    <v-row class="match-height">
        <v-col cols="12">
            <v-card>
                <template #title>
                    <div class="title-and-toolbar d-flex align-center">
                        <span>{{ tt('Budgets') }}</span>
                        <v-btn class="ms-3" color="default" variant="outlined" :disabled="loading" @click="addBudget">{{ tt('Add') }}</v-btn>
                        <v-btn class="ms-3" color="primary" variant="tonal" to="/budgets/report">{{ tt('Budget Report') }}</v-btn>
                        <v-btn density="compact" color="default" variant="text" size="24" class="ms-2" :icon="true" :loading="loading" @click="reload(true)">
                            <template #loader>
                                <v-progress-circular indeterminate size="20"/>
                            </template>
                            <v-icon :icon="mdiRefresh" size="24" />
                            <v-tooltip activator="parent">{{ tt('Refresh') }}</v-tooltip>
                        </v-btn>
                    </div>
                </template>

                <v-table class="table-striped" :hover="!loading">
                    <thead>
                    <tr>
                        <th>{{ tt('Budget Scope') }}</th>
                        <th>{{ tt('Amount') }}</th>
                        <th>{{ tt('Active Months') }}</th>
                        <th class="text-end">{{ tt('Operation') }}</th>
                    </tr>
                    </thead>
                    <tbody v-if="loading">
                    <tr :key="itemIdx" v-for="itemIdx in [1, 2, 3]">
                        <td colspan="4"><v-skeleton-loader type="text" :loading="true" /></td>
                    </tr>
                    </tbody>
                    <tbody v-else-if="!budgets.length">
                    <tr>
                        <td colspan="4">{{ tt('No available budget') }}</td>
                    </tr>
                    </tbody>
                    <tbody v-else>
                    <tr :key="budget.id" v-for="budget in budgets">
                        <td>
                            <div class="d-flex align-center">
                                <span class="budget-color-dot" :style="{ backgroundColor: getBudgetColor(budget) }"></span>
                                <span>{{ getBudgetName(budget) }}</span>
                            </div>
                        </td>
                        <td>{{ formatAmountToLocalizedNumeralsWithCurrency(budget.amount, defaultCurrency) }}</td>
                        <td>{{ getBudgetActiveMonths(budget) }}</td>
                        <td class="text-end">
                            <v-btn class="px-2" color="default" density="comfortable" variant="text" :prepend-icon="mdiPencilOutline" @click="editBudget(budget)">{{ tt('Edit') }}</v-btn>
                            <v-btn class="px-2" color="default" density="comfortable" variant="text" :prepend-icon="mdiDeleteOutline" @click="deleteBudget(budget)">{{ tt('Delete') }}</v-btn>
                        </td>
                    </tr>
                    </tbody>
                </v-table>
            </v-card>
        </v-col>
    </v-row>

    <v-dialog width="520" v-model="showEditDialog">
        <v-card>
            <v-card-title>{{ editBudgetId ? tt('Edit Budget') : tt('Add Budget') }}</v-card-title>
            <v-card-text>
                <v-select item-title="name" item-value="value" :label="tt('Budget Scope')" :items="budgetTypes" v-model="editingBudget.type" />
                <v-select class="mt-2" item-title="name" item-value="id" :label="tt('Transaction Category')" :items="budgetCategories" v-model="editingBudget.categoryId" v-if="editingBudget.type !== BudgetType.TotalExpense" />
                <amount-input class="mt-2" :label="tt('Monthly Budget')" :currency="defaultCurrency" v-model="editingBudget.amount" />
                <v-row class="mt-2">
                    <v-col cols="12" md="6">
                        <v-text-field type="number" :label="tt('Start Month')" :placeholder="tt('YYYYMM, optional')" v-model.number="editingBudget.startYearMonth" />
                    </v-col>
                    <v-col cols="12" md="6">
                        <v-text-field type="number" :label="tt('End Month')" :placeholder="tt('YYYYMM, optional')" v-model.number="editingBudget.endYearMonth" />
                    </v-col>
                </v-row>
            </v-card-text>
            <v-card-actions>
                <v-spacer />
                <v-btn variant="text" @click="showEditDialog = false">{{ tt('Cancel') }}</v-btn>
                <v-btn color="primary" :loading="submitting" @click="saveBudget">{{ tt('Save') }}</v-btn>
            </v-card-actions>
        </v-card>
    </v-dialog>

    <snack-bar ref="snackbar" />
</template>

<script setup lang="ts">
import SnackBar from '@/components/desktop/SnackBar.vue';
import AmountInput from '@/components/desktop/AmountInput.vue';

import { ref, computed, useTemplateRef } from 'vue';

import { useI18n } from '@/locales/helpers.ts';
import { useBudgetsStore } from '@/stores/budget.ts';
import { useTransactionCategoriesStore } from '@/stores/transactionCategory.ts';
import { useUserStore } from '@/stores/user.ts';

import { CategoryType } from '@/core/category.ts';
import { Budget, BudgetType } from '@/models/budget.ts';

import { generateRandomUUID } from '@/lib/misc.ts';

import { mdiDeleteOutline, mdiPencilOutline, mdiRefresh } from '@mdi/js';

type SnackBarType = InstanceType<typeof SnackBar>;

const { tt, formatAmountToLocalizedNumeralsWithCurrency } = useI18n();
const budgetsStore = useBudgetsStore();
const categoriesStore = useTransactionCategoriesStore();
const userStore = useUserStore();
const snackbar = useTemplateRef<SnackBarType>('snackbar');

const loading = ref<boolean>(false);
const submitting = ref<boolean>(false);
const showEditDialog = ref<boolean>(false);
const editBudgetId = ref<string>('');
const editingBudget = ref<Budget>(Budget.createNewBudget());

const defaultCurrency = computed<string>(() => userStore.currentUserDefaultCurrency);
const budgets = computed<Budget[]>(() => budgetsStore.allBudgets);

const budgetTypes = computed(() => [
    { name: tt('Total Spend'), value: BudgetType.TotalExpense },
    { name: tt('Primary Category'), value: BudgetType.PrimaryCategory },
    { name: tt('Subcategory'), value: BudgetType.SecondaryCategory }
]);

const budgetCategories = computed(() => {
    const result: { id: string, name: string }[] = [];
    const expenseCategories = categoriesStore.allTransactionCategories[CategoryType.Expense] || [];

    for (const category of expenseCategories) {
        if (editingBudget.value.type === BudgetType.PrimaryCategory) {
            result.push({ id: category.id, name: category.name });
        } else if (editingBudget.value.type === BudgetType.SecondaryCategory && category.subCategories) {
            for (const subCategory of category.subCategories) {
                result.push({ id: subCategory.id, name: `${category.name} / ${subCategory.name}` });
            }
        }
    }

    return result;
});

function getBudgetName(budget: Budget): string {
    if (budget.type === BudgetType.TotalExpense) {
        return tt('Total Spend');
    }

    const category = categoriesStore.allTransactionCategoriesMap[budget.categoryId];
    const parentCategory = category && category.parentId !== '0' ? categoriesStore.allTransactionCategoriesMap[category.parentId] : undefined;
    return parentCategory ? `${parentCategory.name} / ${category?.name}` : (category?.name || tt('Unknown'));
}

function getBudgetColor(budget: Budget): string {
    if (budget.type === BudgetType.TotalExpense) {
        return 'rgb(var(--v-theme-primary))';
    }

    const category = categoriesStore.allTransactionCategoriesMap[budget.categoryId];
    const parentCategory = category && category.parentId !== '0' ? categoriesStore.allTransactionCategoriesMap[category.parentId] : category;
    return parentCategory ? `#${parentCategory.color}` : 'rgb(var(--v-theme-primary))';
}

function getBudgetActiveMonths(budget: Budget): string {
    if (!budget.startYearMonth && !budget.endYearMonth) {
        return tt('Every month');
    }

    return `${budget.startYearMonth || tt('Any')} - ${budget.endYearMonth || tt('No end')}`;
}

function addBudget(): void {
    editBudgetId.value = '';
    editingBudget.value = Budget.createNewBudget();
    showEditDialog.value = true;
}

function editBudget(budget: Budget): void {
    editBudgetId.value = budget.id;
    editingBudget.value = Budget.of(budget);
    showEditDialog.value = true;
}

function saveBudget(): void {
    if (editingBudget.value.type === BudgetType.TotalExpense) {
        editingBudget.value.categoryId = '0';
    }

    submitting.value = true;
    budgetsStore.saveBudget({ budget: editingBudget.value, isEdit: !!editBudgetId.value, clientSessionId: generateRandomUUID() }).then(() => {
        submitting.value = false;
        showEditDialog.value = false;
    }).catch(error => {
        submitting.value = false;
        if (!error.processed) {
            snackbar.value?.showError(error);
        }
    });
}

function deleteBudget(budget: Budget): void {
    if (!window.confirm(tt('Are you sure you want to delete this budget?'))) {
        return;
    }

    budgetsStore.deleteBudget({ budget }).catch(error => {
        if (!error.processed) {
            snackbar.value?.showError(error);
        }
    });
}

function reload(force: boolean): void {
    loading.value = true;
    Promise.all([
        categoriesStore.loadAllCategories({ force: false }),
        budgetsStore.loadAllBudgets({ force })
    ]).then(() => {
        loading.value = false;
    }).catch(error => {
        loading.value = false;
        if (!error.isUpToDate && !error.processed) {
            snackbar.value?.showError(error);
        }
    });
}

reload(false);
</script>

<style scoped>
.budget-color-dot {
    width: 0.75rem;
    height: 0.75rem;
    border-radius: 999px;
    margin-inline-end: 0.5rem;
}
</style>
