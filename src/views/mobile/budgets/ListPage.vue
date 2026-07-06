<template>
    <f7-page ptr @ptr:refresh="reload">
        <f7-navbar :title="tt('Budgets')" :back-link="tt('Back')">
            <f7-nav-right>
                <f7-link @click="openAddPopup">{{ tt('Add') }}</f7-link>
            </f7-nav-right>
        </f7-navbar>

        <f7-block class="no-margin-bottom">
            <f7-button fill href="/budgets/report">{{ tt('Budget Report') }}</f7-button>
        </f7-block>

        <f7-list strong inset dividers class="margin-top" v-if="loading">
            <f7-list-item :key="itemIdx" title="Budget" after="0.00" v-for="itemIdx in [1, 2, 3]"></f7-list-item>
        </f7-list>

        <f7-block strong inset class="margin-top" v-else-if="!budgets.length">
            {{ tt('No available budget') }}
        </f7-block>

        <f7-list strong inset dividers class="margin-top" v-else>
            <f7-list-item :key="budget.id" link="#" @click="editBudget(budget)" v-for="budget in budgets">
                <template #title>
                    <span class="budget-color-dot" :style="{ backgroundColor: getBudgetColor(budget) }"></span>
                    <span>{{ getBudgetName(budget) }}</span>
                </template>
                <template #after>
                    {{ formatAmountToLocalizedNumeralsWithCurrency(budget.amount, defaultCurrency) }}
                </template>
                <template #footer>
                    {{ getBudgetActiveMonths(budget) }}
                </template>
            </f7-list-item>
        </f7-list>

        <f7-popup v-model:opened="showEditPopup">
            <f7-page>
                <f7-navbar :title="editBudgetId ? tt('Edit Budget') : tt('Add Budget')">
                    <f7-nav-left>
                        <f7-link popup-close>{{ tt('Cancel') }}</f7-link>
                    </f7-nav-left>
                    <f7-nav-right>
                        <f7-link :class="{ disabled: submitting }" @click="saveBudget">{{ tt('Save') }}</f7-link>
                    </f7-nav-right>
                </f7-navbar>
                <f7-list strong inset dividers>
                    <f7-list-input type="select" :label="tt('Budget Scope')" v-model:value="editingBudget.type">
                        <option :value="BudgetType.TotalExpense">{{ tt('Total Spend') }}</option>
                        <option :value="BudgetType.PrimaryCategory">{{ tt('Primary Category') }}</option>
                        <option :value="BudgetType.SecondaryCategory">{{ tt('Subcategory') }}</option>
                    </f7-list-input>
                    <f7-list-input type="select" :label="tt('Transaction Category')" v-model:value="editingBudget.categoryId" v-if="Number(editingBudget.type) !== BudgetType.TotalExpense">
                        <option :key="category.id" :value="category.id" v-for="category in budgetCategories">{{ category.name }}</option>
                    </f7-list-input>
                    <f7-list-item link="#" no-chevron :title="formatAmountToLocalizedNumeralsWithCurrency(editingBudget.amount, defaultCurrency)" :header="tt('Monthly Budget')" @click="showAmountSheet = true">
                        <number-pad-sheet :min-value="1" :max-value="99999999999" :currency="defaultCurrency" v-model:show="showAmountSheet" v-model="editingBudget.amount" />
                    </f7-list-item>
                    <f7-list-input type="number" :label="tt('Start Month')" :placeholder="tt('YYYYMM, optional')" v-model:value="editingBudget.startYearMonth"></f7-list-input>
                    <f7-list-input type="number" :label="tt('End Month')" :placeholder="tt('YYYYMM, optional')" v-model:value="editingBudget.endYearMonth"></f7-list-input>
                </f7-list>
                <f7-list strong inset v-if="editBudgetId">
                    <f7-list-button color="red" @click="deleteCurrentBudget">{{ tt('Delete Budget') }}</f7-list-button>
                </f7-list>
            </f7-page>
        </f7-popup>
    </f7-page>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import type { Router } from 'framework7/types';

import { useI18n } from '@/locales/helpers.ts';
import { useI18nUIComponents } from '@/lib/ui/mobile.ts';

import { useBudgetsStore } from '@/stores/budget.ts';
import { useTransactionCategoriesStore } from '@/stores/transactionCategory.ts';
import { useUserStore } from '@/stores/user.ts';

import { CategoryType } from '@/core/category.ts';
import { Budget, BudgetType } from '@/models/budget.ts';

import { generateRandomUUID } from '@/lib/misc.ts';

defineProps<{
    f7router: Router.Router;
}>();

const { tt, formatAmountToLocalizedNumeralsWithCurrency } = useI18n();
const { showToast, showConfirm } = useI18nUIComponents();
const budgetsStore = useBudgetsStore();
const categoriesStore = useTransactionCategoriesStore();
const userStore = useUserStore();

const loading = ref<boolean>(false);
const submitting = ref<boolean>(false);
const showEditPopup = ref<boolean>(false);
const showAmountSheet = ref<boolean>(false);
const editBudgetId = ref<string>('');
const editingBudget = ref<Budget>(Budget.createNewBudget());

const defaultCurrency = computed<string>(() => userStore.currentUserDefaultCurrency);
const budgets = computed<Budget[]>(() => budgetsStore.allBudgets);

const budgetCategories = computed(() => {
    const result: { id: string, name: string }[] = [];
    const expenseCategories = categoriesStore.allTransactionCategories[CategoryType.Expense] || [];

    for (const category of expenseCategories) {
        if (Number(editingBudget.value.type) === BudgetType.PrimaryCategory) {
            result.push({ id: category.id, name: category.name });
        } else if (Number(editingBudget.value.type) === BudgetType.SecondaryCategory && category.subCategories) {
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
        return 'var(--f7-theme-color)';
    }

    const category = categoriesStore.allTransactionCategoriesMap[budget.categoryId];
    const parentCategory = category && category.parentId !== '0' ? categoriesStore.allTransactionCategoriesMap[category.parentId] : category;
    return parentCategory ? `#${parentCategory.color}` : 'var(--f7-theme-color)';
}

function getBudgetActiveMonths(budget: Budget): string {
    if (!budget.startYearMonth && !budget.endYearMonth) {
        return tt('Every month');
    }

    return `${budget.startYearMonth || tt('Any')} - ${budget.endYearMonth || tt('No end')}`;
}

function openAddPopup(): void {
    editBudgetId.value = '';
    editingBudget.value = Budget.createNewBudget();
    showEditPopup.value = true;
}

function editBudget(budget: Budget): void {
    editBudgetId.value = budget.id;
    editingBudget.value = Budget.of(budget);
    showEditPopup.value = true;
}

function saveBudget(): void {
    editingBudget.value.type = Number(editingBudget.value.type) as BudgetType;
    editingBudget.value.startYearMonth = Number(editingBudget.value.startYearMonth) || 0;
    editingBudget.value.endYearMonth = Number(editingBudget.value.endYearMonth) || 0;

    if (editingBudget.value.type === BudgetType.TotalExpense) {
        editingBudget.value.categoryId = '0';
    }

    submitting.value = true;
    budgetsStore.saveBudget({ budget: editingBudget.value, isEdit: !!editBudgetId.value, clientSessionId: generateRandomUUID() }).then(() => {
        submitting.value = false;
        showEditPopup.value = false;
    }).catch(error => {
        submitting.value = false;
        if (!error.processed) {
            showToast(error.message || error);
        }
    });
}

function deleteCurrentBudget(): void {
    showConfirm('Are you sure you want to delete this budget?', () => {
        budgetsStore.deleteBudget({ budget: editingBudget.value }).then(() => {
            showEditPopup.value = false;
        }).catch(error => {
            if (!error.processed) {
                showToast(error.message || error);
            }
        });
    });
}

function reload(done?: () => void): void {
    loading.value = true;
    Promise.all([
        categoriesStore.loadAllCategories({ force: false }),
        budgetsStore.loadAllBudgets({ force: true })
    ]).then(() => {
        loading.value = false;
        done?.();
    }).catch(error => {
        loading.value = false;
        done?.();
        if (!error.isUpToDate && !error.processed) {
            showToast(error.message || error);
        }
    });
}

reload();
</script>

<style scoped>
.budget-color-dot {
    display: inline-block;
    width: 0.75rem;
    height: 0.75rem;
    border-radius: 999px;
    margin-inline-end: 0.5rem;
}
</style>
