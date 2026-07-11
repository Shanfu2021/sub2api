export default {
  enterpriseManagement: {
    employees: {
      subtitle: 'Balances, concurrency, and RPM assigned to the enterprise can be distributed to employees.',
      create: 'Create employee',
      created: 'Employee created',
      createFailed: 'Failed to create employee',
      deleted: 'Employee deleted. Balance and allocation were returned.',
      deleteTitle: 'Delete employee',
      deleteConfirm: "Delete employee '{email}'? Balance and allocation will be returned to the enterprise.",
      deleteFailed: 'Failed to delete employee',
      groupsTitle: 'Employee groups: {email}',
      emptyGroups: 'No exclusive groups are available for employees',
      groupSaved: 'Employee group saved',
      groupRemoved: 'Employee group removed',
      groupFailed: 'Failed to save employee group',
      defaultGroups: {
        title: 'Default employee groups',
        description: 'Selected groups are automatically assigned after employees are created or imported. Rates use the enterprise current effective rates; enterprises cannot reprice them.',
        saved: 'Default employee group saved',
        removed: 'Default employee group removed',
        failed: 'Failed to save default employee group'
      },
      insufficientAllocation: 'Remaining concurrency or RPM is not enough to allocate',
      loadFailed: 'Failed to load employees',
      import: {
        title: 'Import employees',
        file: 'JSON file',
        formatTitle: 'Import format: each employee record must include exactly these required fields: email, username, password, concurrency, rpm',
        skipHint: 'Import does not allocate balance. Imported employees start with balance 0. Records missing any required field or containing invalid fields are skipped.',
        validRows: 'File records',
        requiredQuota: 'Estimated quota',
        row: 'Row',
        reason: 'Reason',
        submit: 'Import',
        importing: 'Importing...',
        result: 'Import complete: created {created}, skipped {skipped}',
        failed: 'Failed to import employees',
        invalidJson: 'Failed to parse JSON file',
        invalidJsonShape: 'JSON must be an employee array or an object with an employees array',
        emptyFile: 'No employee records were found in the file',
        quotaExceeded: 'Enterprise quota is not enough: current concurrency {currentConcurrency}, required {requiredConcurrency}; current RPM {currentRpm}, required {requiredRpm}'
      },
      balanceInit: {
        title: 'Initialize balances',
        target: 'Set every employee balance to',
        hint: 'This sets all direct employee balances to the same value. Increases are deducted from the enterprise balance; decreases are returned to the enterprise.',
        submit: 'Initialize',
        success: 'Initialized {count} employee balances to {balance}. Net required balance: {required}',
        failed: 'Failed to initialize employee balances',
        exceeded: 'Enterprise balance is not enough: current {current}, required {required}'
      }
    },
    groups: {
      subtitle: 'View public and exclusive groups assigned to this enterprise. Only effective enterprise rates are shown.',
      loadFailed: 'Failed to load enterprise groups'
    }
  },
}
